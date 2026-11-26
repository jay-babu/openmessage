const nativeHost = "com.openmessage.google_refresh";
const refreshAlarm = "openmessage-refresh";
const messagesURL = "https://messages.google.com/web/";
const accountURL = "https://google.com/";
const accountDomain = ".google.com";
const requiredCookies = ["SID", "HSID", "SSID", "APISID", "SAPISID"];
let publishing;

async function publishLogin() {
  let { profile } = await chrome.storage.local.get("profile");
  if (!profile) {
    profile = crypto.randomUUID();
    await chrome.storage.local.set({ profile });
  }

  async function accountCookie(name) {
    const cookie = await chrome.cookies.get({ name, url: accountURL });
    if (cookie?.value && (cookie.domain === accountDomain || cookie.domain === "google.com")) {
      return cookie.value;
    }
    const candidates = await chrome.cookies.getAll({ name, domain: "google.com" });
    return candidates.find(({ domain, path }) => domain === accountDomain && path === "/")?.value;
  }

  const entries = await Promise.all(requiredCookies.map(async (name) => [name, await accountCookie(name)]));
  const missing = entries.filter(([, value]) => !value).map(([name]) => name);
  let cookies = null;
  if (missing.length === 0) {
    cookies = Object.fromEntries(entries);
    const optional = await Promise.all([
      chrome.cookies.get({ name: "OSID", url: messagesURL }).then((cookie) => ["OSID", cookie?.value]),
      chrome.cookies.get({ name: "__Secure-OSID", url: messagesURL }).then((cookie) => ["__Secure-OSID", cookie?.value]),
      accountCookie("__Secure-1PSIDTS").then((value) => ["__Secure-1PSIDTS", value]),
    ]);
    for (const [name, value] of optional) {
      if (value) cookies[name] = value;
    }
  }

  const response = await chrome.runtime.sendNativeMessage(nativeHost, { profile, cookies });
  if (!response?.ok) {
    throw new Error(response?.error || "Could not update OpenMessage.");
  }
  await chrome.action.setBadgeText({ text: cookies ? "ON" : "!" });
  await chrome.action.setTitle({
    title: cookies ? "OpenMessage login refresh is active" : "OpenMessage login missing: " + missing.join(", "),
  });
}

function refresh() {
  if (publishing) return;
  publishing = publishLogin()
    .catch(async (error) => {
      await chrome.action.setBadgeText({ text: "!" });
      await chrome.action.setTitle({ title: "OpenMessage refresh: " + error.message });
    })
    .finally(() => { publishing = undefined; });
}

function start() {
  chrome.alarms.create(refreshAlarm, { periodInMinutes: 1 });
  refresh();
}

chrome.runtime.onInstalled.addListener(start);
chrome.runtime.onStartup.addListener(start);
chrome.action.onClicked.addListener(start);
chrome.alarms.onAlarm.addListener(({ name }) => {
  if (name === refreshAlarm) refresh();
});
chrome.cookies.onChanged.addListener(({ cookie, cause }) => {
  if (cause === "overwrite") return;
  if ((cookie.domain.replace(/^\./, "") === "google.com" &&
       (requiredCookies.includes(cookie.name) || cookie.name === "__Secure-1PSIDTS")) ||
      (cookie.domain.replace(/^\./, "") === "messages.google.com" &&
       ["OSID", "__Secure-OSID"].includes(cookie.name))) {
    refresh();
  }
});
