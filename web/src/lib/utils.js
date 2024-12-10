export function copyToClipboard() {
  navigator.clipboard.writeText(apiToken);
  copied = true;
  setTimeout(() => {
    copied = false;
  }, 2000);
}