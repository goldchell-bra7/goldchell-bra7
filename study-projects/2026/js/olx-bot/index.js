const fs = require('fs');
const axios = require('axios');
const cheerio = require('cheerio');
let seenLinks = new Set()

// Telegram API
const BOT_TOKEN = process.env.BOT_TOKEN;
const CHAT_ID = process.env.CHAT_ID;

// Settings
const targetPrice = 12000;
const maxPercentDiff = 0.3;

const keywords = ["сдам", "квартиру", "2к", "2-к", "кв", "двухкомнатную", "двухкомнатная", "3к", "3-к", "трёхкомнатную", "трёхкомнатная", "евроремонт", "харьков", "Харків", "здам", "грн"];
const excludeKeywords = ["сутки", "1к", "гараж", "гостинка", "салтовка", "салтівка", "героїв праці", "героев труда", "студенческая", "студентська",];

// Price parsing
function parsePrice(text) {
    if (!text) return 0;

    const match = text
        .replace(/\s/g, '')
        .match(/\d+/g);

    if (!match) return 0;

    return parseInt(match[0], 10);
}
function extractId(link) {
    const match = link.match(/ID([a-zA-Z0-9]+)/);
    if (!match) return 0;

    return parseInt(match[1], 36);
}
function loadSeen() {
    try {
        if (fs.existsSync('seen.json')) {
            const data = JSON.parse(fs.readFileSync('seen.json'));
            seenLinks = new Set(data);
            console.log('📂 Загружено объявлений:', seenLinks.size);
        }
    } catch (e) {
        console.log('❌ Ошибка загрузки seen');
    }
}

function saveSeen() {
    try {
        fs.writeFileSync('seen.json', JSON.stringify([...seenLinks]));
    } catch (e) {
        console.log('❌ Ошибка сохранения seen');
    }
}

function saveLastId() {
    try {
        fs.writeFileSync('lastId.json', JSON.stringify({ lastMaxId }));
        console.log('💾 Сохранён lastMaxId:', lastMaxId);
    } catch (err) {
        console.error('❌ Ошибка сохранения ID:', err.message);
    }
}

// Filters
function isValidAd(title, price) {
    const lower = title.toLowerCase();

    const hasKeyword = keywords.some(k => lower.includes(k));
    const hasExclude = excludeKeywords.some(k => lower.includes(k));

    const diff = Math.abs(price - targetPrice);
    const threshold = targetPrice * maxPercentDiff;

    return hasKeyword && !hasExclude && diff <= threshold;
}

// Telegram message
async function sendToTelegram(ad) {
    try {
        const res = await axios.post(`https://api.telegram.org/bot${BOT_TOKEN}/sendMessage`, {
            chat_id: CHAT_ID,
             text: `🔥 Новое объявление!

📦 ${ad.title}
💰 ${ad.price} грн
🔗 ${ad.link}`
        });

        if (!res.data.ok) {
            console.error('❌ Telegram error:', res.data);
        } else {
            console.log('✅ Отправлено:', ad.title);
        }
    } catch (err) {
        console.error('❌ Telegram:', err.response?.data || err.message);
    }
}

// Site parsing
async function checkOLX(isFirstRun = false) {
    try {
        const url = 'https://www.olx.ua/uk/nedvizhimost/kvartiry/dolgosrochnaya-arenda-kvartir/kharkov/?search[order]=created_at:desc';

        const { data } = await axios.get(url, {
    headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
        'Accept-Language': 'ru-RU,ru;q=0.9'
    }
});

        const $ = cheerio.load(data);
        console.log('Найдено объявлений:', $('[data-cy="l-card"]').length);

        const cards = $('[data-cy="l-card"]').toArray().slice(0,15)

for (const el of cards) {
    
    console.log('👉 Зашёл в цикл');

    let title = $(el).find('a').text().trim();
    
    console.log('📦 RAW TITLE:', title);

    title = title.replace(/\s+/g, ' ').trim();

    const priceText = $(el).find('[data-testid="ad-price"]').text().trim();

    let link = $(el).find('a').attr('href');
    link = link.split('?')[0];
    if (link && !link.startsWith('http')) {
        link = 'https://www.olx.ua' + link;
    }

    if (!link) continue;

    if (seenLinks.has(link)) {
        console.log('⏭ ПРОПУСК (seen):', link);
    continue;
    }

    const price = parsePrice(priceText);

    console.log('🔎 Проверка:', title, price);

    if (isValidAd(title, price)) {
        console.log('✅ ПРОШЕЛ ФИЛЬТР:', title);

        for (const word of excludeKeywords) {
    if (title.toLowerCase().includes(word)) {
        console.log('⛔ exclude слово:', word);
        return false;
    }
        }

        console.log('👉 VALIDATE:', title, price);
        
        if (!isFirstRun) {
    seenLinks.add(link);
    continue;
        }
        } else {
    console.log('❌ НЕ ПРОШЕЛ:', title);
    }
            console.log('📤 ОТПРАВКА:', title);
            await sendToTelegram({ title, price, link });
    }
        seenLinks.add(link);
saveSeen();
        }
    }

} catch (err) {
        console.error('❌ Ошибка парсинга:', err.message);
    }
}

// Random interval
function getRandomDelay() {
    return Math.floor(Math.random() * (30000 - 15000 + 1)) + 15000;
}

// Cycle start
async function start() {
    console.log('🚀 START ЗАПУСТИЛСЯ');

    loadSeen();

    console.log('📥 Перед первым checkOLX');
    await checkOLX(true);
    console.log('✅ После первого checkOLX');

    await new Promise(r => setTimeout(r, 3000));

    while (true) {
        console.log('🔍 Проверка OLX...');
        
        await checkOLX();

        const delay = getRandomDelay();
        console.log(`⏳ Ждём ${Math.round(delay / 1000)} сек`);

        await new Promise(r => setTimeout(r, delay));
    }
}

start();
console.log('🚀 ФАЙЛ ЗАПУЩЕН');
