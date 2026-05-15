import { browser, output } from "kit";

const before = await browser.list();

output({
  listed: Array.isArray(before),
  count: before.length,
  hasLaunch: typeof browser.launch === "function",
  hasClose: typeof browser.close === "function",
});
