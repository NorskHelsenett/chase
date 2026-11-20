/** @type {Record<string, string>} */
const HTML_ESCAPES = {
	'&': '&amp;',
	'<': '&lt;',
	'>': '&gt;',
	'"': '&quot;',
	"'": '&#39;'
};

/**
 * Escape a value for safe interpolation into an HTML string
 * (text content or a quoted attribute value).
 * @param {unknown} value
 * @returns {string}
 */
export function escapeHtml(value) {
	if (value === null || value === undefined) return '';
	return String(value).replace(/[&<>"']/g, (ch) => HTML_ESCAPES[ch]);
}
