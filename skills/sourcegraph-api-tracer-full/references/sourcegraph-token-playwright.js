/**
 * Sourcegraph Access Token 自动生成脚本（Node.js + Playwright）
 *
 * 配置源：同目录 config.json（所有参数从此读取，新 Token 也写回此文件）。
 *
 * 使用方法:
 *   node sourcegraph-token-playwright.js [USERNAME] [PASSWORD] [TOKEN_NAME] [EXPIRES_AT]
 *   node sourcegraph-token-playwright.js               # 读 config.json 中的 LDAP 账号
 *   node sourcegraph-token-playwright.js my-token-name # 只指定 Token 名称，账号来自 config.json
 */

const fs = require('fs');
const path = require('path');
const readline = require('readline');
const { chromium } = require('playwright');

const CONFIG_JSON_PATH = path.join(__dirname, 'config.json');

// ---------------------------------------------------------------------------
// config.json 读写
// ---------------------------------------------------------------------------

function readConfigJson() {
  try {
    const raw = fs.readFileSync(CONFIG_JSON_PATH, 'utf8');
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

function writeTokenToConfigJson(token, url) {
  let data = readConfigJson();
  data['SOURCEGRAPH_TOKEN'] = token;
  if (url) data['SOURCEGRAPH_URL'] = url.replace(/\/+$/, '');
  try {
    const tmp = CONFIG_JSON_PATH + '.tmp';
    fs.writeFileSync(tmp, JSON.stringify(data, null, 2) + '\n', 'utf8');
    fs.renameSync(tmp, CONFIG_JSON_PATH);
    console.log(`✅ Token 已写回: ${CONFIG_JSON_PATH}`);
    return true;
  } catch (e) {
    console.error(`写入 config.json 失败: ${e.message}`);
    return false;
  }
}

// ---------------------------------------------------------------------------
// 从 config.json 读取运行时参数（os.environ 优先于 config.json）
// ---------------------------------------------------------------------------

const _cfg = readConfigJson();

function cfgGet(key, defaultVal = '') {
  if (process.env[key] !== undefined && process.env[key] !== '') {
    return process.env[key];
  }
  const v = _cfg[key];
  return v !== undefined && v !== null ? String(v) : defaultVal;
}

const SOURCEGRAPH_URL = cfgGet('SOURCEGRAPH_URL', 'https://sourcegraph.yc345.tv').replace(/\/+$/, '');
const GITLAB_HOST = cfgGet('SOURCEGRAPH_GITLAB_HOST', 'gitlab.yc345.tv').trim().toLowerCase();
// 匹配 GitLab LDAP 登录页（sign_in 页面含 LDAP tab，或直接跳 ldap/*/auth）
const GITLAB_LDAP_URL_RE = new RegExp(
  `${GITLAB_HOST.replace(/\./g, '\\.')}/(users/sign_in|users/auth/ldap)`
);
const DEBUG_SCREENSHOT = path.join(process.cwd(), 'sourcegraph-token-debug.png');

function playwrightHeadless() {
  const v = cfgGet('PLAYWRIGHT_HEADLESS', '1').trim().toLowerCase();
  return ['1', 'true', 'yes', 'on'].includes(v);
}

function getDefaultExpiry() {
  const d = new Date();
  d.setFullYear(d.getFullYear() + 3);
  return d.toISOString().split('T')[0];
}

const CONFIG = {
  username: '',
  password: '',
  tokenName: 'cli-token',
  expiresAt: getDefaultExpiry(),
};

// ---------------------------------------------------------------------------
// 工具
// ---------------------------------------------------------------------------

async function question(query) {
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise((resolve) => {
    rl.question(query, (answer) => { rl.close(); resolve(answer); });
  });
}

async function waitAndType(page, selector, text, options = {}) {
  const { timeout = 30000, clear = true } = options;
  await page.waitForSelector(selector, { timeout, state: 'visible' });
  if (clear) await page.click(selector, { clickCount: 3 });
  await page.type(selector, text);
}

async function extractToken(page) {
  // 尝试从 input[readonly] 或 input[type=text] 中读取 token 值（截图显示 token 展示在 input 框内）
  const selectors = [
    '.access-token-value',
    '[data-testid="token-value"]',
    '.token-value',
    'input[readonly][value]',
    'input[type="text"][readonly]',
  ];
  for (const sel of selectors) {
    try {
      const el = page.locator(sel).first();
      const visible = await el.isVisible({ timeout: 2000 }).catch(() => false);
      if (!visible) continue;
      // 先尝试 inputValue（input 元素），再尝试 textContent（div/span）
      let val = '';
      try { val = await el.inputValue(); } catch { /* ignore */ }
      if (!val) { try { val = (await el.textContent()) || ''; } catch { /* ignore */ } }
      val = val.trim();
      if (val && /^[a-f0-9]{20,}$/.test(val)) return val;
    } catch { /* ignore */ }
  }
  return null;
}

// ---------------------------------------------------------------------------
// 主流程
// ---------------------------------------------------------------------------

async function main() {
  const args = process.argv.slice(2);

  if (args.length >= 2) {
    CONFIG.username = args[0];
    CONFIG.password = args[1];
    if (args.length >= 3) CONFIG.tokenName = args[2];
    if (args.length >= 4) CONFIG.expiresAt = args[3];
  } else {
    console.log('=== Sourcegraph Token 生成工具 ===\n');
    if (args.length >= 1) CONFIG.tokenName = args[0];

    CONFIG.username = cfgGet('SOURCEGRAPH_LDAP_USERNAME', '').trim();
    if (!CONFIG.username) CONFIG.username = (await question('GitLab LDAP 用户名: ')).trim();

    CONFIG.password = cfgGet('SOURCEGRAPH_LDAP_PASSWORD', '').trim();
    if (!CONFIG.password) CONFIG.password = (await question('GitLab LDAP 密码: ')).trim();
  }

  if (!CONFIG.username || !CONFIG.password) {
    console.error('错误: 用户名或密码为空（请传参或在 config.json 中配置 SOURCEGRAPH_LDAP_*）');
    process.exit(1);
  }

  console.log('\n配置信息:');
  console.log(`  - Sourcegraph: ${SOURCEGRAPH_URL}`);
  console.log(`  - GitLab 主机: ${GITLAB_HOST}`);
  console.log(`  - Token 名称: ${CONFIG.tokenName}`);
  console.log(`  - 过期时间: ${CONFIG.expiresAt}`);
  console.log(`  - Playwright headless: ${playwrightHeadless()}\n`);

  const browser = await chromium.launch({
    headless: playwrightHeadless(),
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  const context = await browser.newContext({ viewport: { width: 1280, height: 720 } });
  const page = await context.newPage();
  page.on('console', (msg) => { if (msg.type() === 'error') console.log(`[Browser] ${msg.text()}`); });

  try {
    console.log('[1/6] 打开 Sourcegraph...');
    await page.goto(SOURCEGRAPH_URL, { waitUntil: 'networkidle', timeout: 60000 });

    console.log('[2/6] 选择 GitLab 登录方式...');
    await page.locator('button:has-text("GitLab"), a:has-text("GitLab")').first().click({ timeout: 10000 });

    console.log('[3/6] 等待 GitLab 登录页...');
    await page.waitForURL(GITLAB_LDAP_URL_RE, { timeout: 30000 });

    console.log('[4/6] 填写 LDAP 凭证...');
    // GitLab LDAP 页：id 可能为 username/spUsername/ldap_user 等，兼容多种
    await waitAndType(page, '#username, [name="username"], input[type="text"]:visible', CONFIG.username);
    await waitAndType(page, '#password, [name="password"], input[type="password"]:visible', CONFIG.password);
    await page.click('button[type="submit"], input[type="submit"]');

    console.log('[5/6] 处理 OAuth 回调...');
    const sgGlob = `${SOURCEGRAPH_URL}/-*`;
    await page.waitForURL(sgGlob, { timeout: 30000 }).catch(() => console.log('  (可能已在 Sourcegraph 页面)'));

    const authBtn = page.locator('button:has-text("Authorize"), input[value="Authorize"]').first();
    if (await authBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      console.log('  授权 GitLab 应用访问...');
      await authBtn.click();
      await page.waitForURL(sgGlob, { timeout: 30000 });
    }

    console.log('[6/6] 创建 Access Token...');
    // 优先用用户自己的 Token 设置页（非管理员也能访问），如果不存在再试 admin 页
    await page.goto(`${SOURCEGRAPH_URL}/user/settings/tokens/new`, { waitUntil: 'networkidle', timeout: 30000 });
    // 若落在 404 或跳转到搜索页，说明路径不对，尝试旧路径
    if (page.url().includes('/search') || page.url().endsWith('/')) {
      await page.goto(`${SOURCEGRAPH_URL}/-/admin/tokens`, { waitUntil: 'networkidle', timeout: 30000 });
    }

    // 若已在 /new 页（表单直接可见），跳过 "Create access token" 按钮
    const createBtn = page.locator('button:has-text("Create access token"), button:has-text("Generate token")').or(page.locator('[data-testid="create-token-btn"]'));
    const createBtnVisible = await createBtn.first().isVisible({ timeout: 3000 }).catch(() => false);
    if (createBtnVisible) {
      await createBtn.first().click({ timeout: 10000 });
    }

    await waitAndType(page, 'input[name="tokenName"], input[placeholder*="token" i], input[id*="name"]', CONFIG.tokenName, { clear: true });

    const expiresInput = page.locator('input[type="date"], input[name*="expir" i], input[id*="expir" i]').first();
    if (await expiresInput.isVisible({ timeout: 5000 }).catch(() => false)) {
      await expiresInput.click({ clickCount: 3 });
      await expiresInput.type(CONFIG.expiresAt);
    }

    await page.locator('button:has-text("Create token"), button:has-text("Generate token"), button[type="submit"]').first().click();
    // 等待 Token 值出现（可能是 input[readonly] 或 .access-token-value 等）
    await page.waitForSelector(
      '.access-token-value, [data-testid="token-value"], .token-value, input[readonly][value]',
      { timeout: 15000 }
    );

    const token = await extractToken(page);
    if (!token) throw new Error('无法提取 Token，请手动复制');

    console.log('\n✅ Token 生成成功!');
    console.log(`\n你的 Access Token:\n${token}\n`);
    writeTokenToConfigJson(token, SOURCEGRAPH_URL);

  } catch (error) {
    console.error(`\n❌ 错误: ${error.message}`);
    try { await page.screenshot({ path: DEBUG_SCREENSHOT }); console.log(`截图: ${DEBUG_SCREENSHOT}`); } catch { /* ignore */ }
    process.exit(1);
  } finally {
    await browser.close();
  }
}

main();
