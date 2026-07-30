package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds the application settings
type Config struct {
	TelegramToken  string
	TelegramChatID string
	OLXURL         string
	Interval       time.Duration
}

type Filter struct {
	MinPrice         float64
	MaxPrice         float64
	RequiredKeywords []string
	ExcludedKeywords []string
	AllowedDistricts []string
}

// CurrentFilter is the active filter configuration.
// Edit this function to apply your desired filters!
func GetActiveFilter() Filter {
	return Filter{
		MinPrice:         5000,
		MaxPrice:         16000,
		RequiredKeywords: []string{"2к", "2-комнатная", "3к", "3-комнатная", "Евроремонт"},
		ExcludedKeywords: []string{"без фото", "1к", "1-комнатная", "Салтовка"},
		AllowedDistricts: []string{"Шевченківський", "Холодногірський"},
	}
}

// Match checks if an offer satisfies the filter criteria
func (f Filter) Match(offer Offer) bool {
	priceVal := offer.GetPriceFloat()

	// 1. Price Filter
	if f.MinPrice > 0 && priceVal < f.MinPrice {
		return false
	}
	if f.MaxPrice > 0 && priceVal > f.MaxPrice {
		return false
	}

	// 2. District (Location) Filter
	if len(f.AllowedDistricts) > 0 {
		matchedDistrict := false
		districtName := strings.ToLower(offer.AreaServed.Name)
		for _, d := range f.AllowedDistricts {
			if strings.Contains(districtName, strings.ToLower(d)) {
				matchedDistrict = true
				break
			}
		}
		if !matchedDistrict {
			return false
		}
	}

	// 3. Required Keywords Filter
	titleLower := strings.ToLower(offer.Name)
	if len(f.RequiredKeywords) > 0 {
		matchedKeyword := false
		for _, kw := range f.RequiredKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				matchedKeyword = true
				break
			}
		}
		if !matchedKeyword {
			return false
		}
	}

	// 4. Excluded Keywords Filter
	for _, kw := range f.ExcludedKeywords {
		if strings.Contains(titleLower, strings.ToLower(kw)) {
			return false
		}
	}

	return true
}

// ==========================================
// STRUCTURED DATA TYPES FOR JSON-LD
// ==========================================

type AreaServed struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

type Offer struct {
	Type          string      `json:"@type"`
	Name          string      `json:"name"`
	Price         interface{} `json:"price"` // Can be float64 or string
	Url           string      `json:"url"`
	PriceCurrency string      `json:"priceCurrency"`
	AreaServed    AreaServed  `json:"areaServed"`
	Image         []string    `json:"image"`
}

// GetPriceFloat helper converts the interface Price field to a float64
func (o Offer) GetPriceFloat() float64 {
	switch p := o.Price.(type) {
	case float64:
		return p
	case int:
		return float64(p)
	case int64:
		return float64(p)
	case string:
		if val, err := strconv.ParseFloat(p, 64); err == nil {
			return val
		}
	}
	return 0
}

type AggregateOffer struct {
	Type   string  `json:"@type"`
	Offers []Offer `json:"offers"`
}

type ProductLD struct {
	Type   string         `json:"@type"`
	Offers AggregateOffer `json:"offers"`
}

// ==========================================
// SEEN LISTINGS CACHE
// ==========================================

type Cache struct {
	mu   sync.RWMutex
	urls map[string]time.Time
}

func NewCache() *Cache {
	return &Cache{
		urls: make(map[string]time.Time),
	}
}

// Add attempts to insert a URL. Returns true if it was new, false if already seen.
func (c *Cache) Add(url string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.urls[url]; exists {
		return false
	}
	// Keep cache size bounded
	if len(c.urls) > 2000 {
		c.urls = make(map[string]time.Time) // simple reset if it grows too large
	}
	c.urls[url] = time.Now()
	return true
}

// ==========================================
// MAIN BOT IMPLEMENTATION
// ==========================================

// Regex to extract application/ld+json contents
var scriptRegexp = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)

func main() {
	log.Println("Starting OLX Parser Bot...")
	loadEnv()

	config := loadConfig()
	cache := NewCache()
	activeFilter := GetActiveFilter()

	// Start a simple HTTP server to satisfy Render's port binding requirement.
	// Render allocates a port dynamically, which is passed in the PORT environment variable.
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080" // Default port if running locally
		}
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OLX Bot is running!"))
		})
		log.Printf("Starting HTTP server on port %s for Render/VPS health check...", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Output loaded config details
	log.Printf("Target URL: %s", config.OLXURL)
	log.Printf("Parsing Interval: %v", config.Interval)
	if config.TelegramToken == "" || config.TelegramChatID == "" {
		log.Println("WARNING: TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is not configured. Running in LOG-ONLY mode.")
	} else {
		log.Println("Telegram credentials detected. Notifications will be sent.")
	}

	isFirstRun := true

	for {
		log.Println("Fetching OLX page...")
		offers, err := fetchOLXOffers(config.OLXURL)
		if err != nil {
			log.Printf("Error fetching OLX offers: %v", err)
		} else {
			newOffersCount := 0
			filteredOffersCount := 0

			// Iterate in reverse (older to newer) so Telegram notifications arrive in chronological order
			for i := len(offers) - 1; i >= 0; i-- {
				offer := offers[i]

				// Try to add to Cache (seen check)
				if cache.Add(offer.Url) {
					newOffersCount++

					// Apply filters
					if activeFilter.Match(offer) {
						if isFirstRun {
							// On first run, we just populate the cache and log, avoiding spamming old ads
							log.Printf("[First Run Cache Population] Found existing listing: %s - %v %s (District: %s)",
								offer.Name, offer.GetPriceFloat(), offer.PriceCurrency, offer.AreaServed.Name)
							continue
						}

						log.Printf("NEW MATCH: %s - %v %s", offer.Name, offer.GetPriceFloat(), offer.PriceCurrency)

						// Format and send notification
						msg := formatMessage(offer)
						if config.TelegramToken != "" && config.TelegramChatID != "" {
							err := sendTelegramMessage(config.TelegramToken, config.TelegramChatID, msg)
							if err != nil {
								log.Printf("Error sending telegram message: %v", err)
							} else {
								log.Println("Notification sent successfully.")
							}
							// Add a brief sleep between sending messages to prevent Telegram rate limit issues
							time.Sleep(500 * time.Millisecond)
						} else {
							log.Printf("[MOCK NOTIFICATION]\n%s\n-------------------", msg)
						}
					} else {
						filteredOffersCount++
					}
				}
			}

			if isFirstRun {
				log.Printf("First run complete. Loaded %d listings into cache. Filtering active.", len(offers))
				isFirstRun = false
			} else {
				log.Printf("Scan complete. New listings found: %d. Filtered out: %d.", newOffersCount, filteredOffersCount)
			}
		}

		time.Sleep(config.Interval)
	}
}

// fetchOLXOffers downloads the page and parses the JSON-LD script block containing offers
func fetchOLXOffers(targetURL string) ([]Offer, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to resemble a real browser request to bypass bot blockages
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "uk-UA,uk;q=0.9,en-US;q=0.8,en;q=0.7,ru;q=0.6")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlContent := string(bodyBytes)

	// Extract JSON-LD script contents using regex
	matches := scriptRegexp.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		rawJSON := strings.TrimSpace(match[1])

		// Quick check if this is the Product structured data block
		if strings.Contains(rawJSON, `"@type":"Product"`) || strings.Contains(rawJSON, `"@type": "Product"`) {
			var product ProductLD
			if err := json.Unmarshal([]byte(rawJSON), &product); err != nil {
				// Sometimes JSON contains unescaped controls, let's log and try to proceed
				continue
			}

			if product.Type == "Product" && len(product.Offers.Offers) > 0 {
				return product.Offers.Offers, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find offers in page's JSON-LD script blocks")
}

// formatMessage creates an HTML-formatted message for Telegram
func formatMessage(offer Offer) string {
	price := offer.GetPriceFloat()
	priceStr := fmt.Sprintf("%.0f", price)
	if price == 0 {
		priceStr = "Не вказана"
	}

	district := offer.AreaServed.Name
	if district == "" {
		district = "Не вказаний"
	}

	var builder strings.Builder
	builder.WriteString("<b>🆕 Нове оголошення на OLX!</b>\n\n")
	builder.WriteString(fmt.Sprintf("📌 <b>Назва:</b> %s\n", escapeHTML(offer.Name)))
	builder.WriteString(fmt.Sprintf("💵 <b>Ціна:</b> %s %s\n", priceStr, offer.PriceCurrency))
	builder.WriteString(fmt.Sprintf("📍 <b>Район:</b> %s\n\n", escapeHTML(district)))
	builder.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">Переглянути оголошення</a>", offer.Url))

	return builder.String()
}

// escapeHTML replaces basic HTML characters with their HTML entities to avoid broken Telegram messages
func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}

// sendTelegramMessage posts the message to Telegram HTTP API
func sendTelegramMessage(token, chatID, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)
	data.Set("parse_mode", "HTML")
	data.Set("disable_web_page_preview", "false")

	// Set timeout for telegram post
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// loadEnv loads variables from a .env file into the environment if it exists
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		// File does not exist, which is fine
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

// loadConfig reads environment variables and supplies defaults if missing
func loadConfig() Config {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	olxURL := os.Getenv("OLX_URL")
	if olxURL == "" {
		olxURL = "https://www.olx.ua/uk/nedvizhimost/kvartiry/dolgosrochnaya-arenda-kvartir/kharkov/"
	}

	intervalStr := os.Getenv("PARSING_INTERVAL")
	interval := 30 * time.Second // Default interval
	if intervalStr != "" {
		if parsed, err := time.ParseDuration(intervalStr); err == nil {
			// Require at least 5 seconds to avoid heavy load
			if parsed >= 5*time.Second {
				interval = parsed
			}
		}
	}

	return Config{
		TelegramToken:  token,
		TelegramChatID: chatID,
		OLXURL:         olxURL,
		Interval:       interval,
	}
}
