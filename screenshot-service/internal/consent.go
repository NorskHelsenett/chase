package internal

import (
	"github.com/go-rod/rod"
)

// ConsentRemover handles cookie consent banner removal
type ConsentRemover struct {
	// Common consent banner selectors
	selectors []string
}

// NewConsentRemover creates a new consent remover
func NewConsentRemover() *ConsentRemover {
	return &ConsentRemover{
		selectors: []string{
			// Common cookie consent frameworks
			"#onetrust-banner-sdk",
			"#onetrust-consent-sdk",
			".onetrust-pc-dark-filter",
			"#CybotCookiebotDialog",
			".cc-banner",
			".cc-window",
			".cookie-consent",
			".cookie-banner",
			".cookie-notice",
			".gdpr-banner",
			".privacy-notice",

			// Specific platforms
			".fc-consent-root", // Federated Consent
			"#truste-consent-track",
			".trustarc-banner",
			".qc-cmp2-container", // Quantcast
			"[id^='sp_message_container']", // SourcePoint
			".didomi-popup",
			".osano-cm-widget",

			// Generic patterns
			"[class*='cookie']",
			"[id*='cookie']",
			"[class*='consent']",
			"[id*='consent']",
			"[class*='gdpr']",
			"[id*='gdpr']",
			"[aria-label*='cookie']",
			"[aria-label*='consent']",

			// Overlay patterns
			".modal-backdrop",
			".overlay",
			"[class*='overlay']",
		},
	}
}

// RemoveConsent removes cookie consent banners from the page
func (c *ConsentRemover) RemoveConsent(page *rod.Page) error {
	// JavaScript to remove consent banners and overlays
	js := `(() => {
		try {
			// Common consent banner selectors
			const selectors = [
				'#onetrust-banner-sdk',
				'#onetrust-consent-sdk',
				'.onetrust-pc-dark-filter',
				'#CybotCookiebotDialog',
				'.cc-banner',
				'.cc-window',
				'.cookie-consent',
				'.cookie-banner',
				'.cookie-notice',
				'.gdpr-banner',
				'.privacy-notice',
				'.fc-consent-root',
				'#truste-consent-track',
				'.trustarc-banner',
				'.qc-cmp2-container',
				'[id^="sp_message_container"]',
				'.didomi-popup',
				'.osano-cm-widget'
			];

			// Remove elements matching selectors
			selectors.forEach(selector => {
				try {
					document.querySelectorAll(selector).forEach(el => el.remove());
				} catch (e) {
					// Skip invalid selectors
				}
			});

			// Remove elements with cookie/consent/gdpr in class or id
			const removeByPattern = () => {
				const elements = Array.from(document.querySelectorAll('*'));
				elements.forEach(el => {
					try {
						const classStr = (el.className && el.className.toString ? el.className.toString() : '').toLowerCase();
						const idStr = (el.id || '').toLowerCase();

						const hasCookieTerms = classStr.includes('cookie') || classStr.includes('consent') ||
											   classStr.includes('gdpr') || idStr.includes('cookie') ||
											   idStr.includes('consent') || idStr.includes('gdpr');

						const hasUITerms = classStr.includes('banner') || classStr.includes('notice') ||
										   classStr.includes('modal') || classStr.includes('popup') ||
										   classStr.includes('dialog');

						if (hasCookieTerms && hasUITerms) {
							el.remove();
						}
					} catch (e) {
						// Skip elements that can't be processed
					}
				});
			};
			removeByPattern();

			// Remove modal backdrops and overlays
			document.querySelectorAll('.modal-backdrop, .overlay, [class*="overlay"]').forEach(el => {
				try {
					const style = window.getComputedStyle(el);
					if (style.position === 'fixed' && parseInt(style.zIndex) > 1000) {
						el.remove();
					}
				} catch (e) {
					// Skip if can't get computed style
				}
			});

			// Re-enable scrolling (some consent banners disable it)
			if (document.body) {
				document.body.style.overflow = '';
				document.body.style.removeProperty('overflow');
			}
			if (document.documentElement) {
				document.documentElement.style.overflow = '';
				document.documentElement.style.removeProperty('overflow');
			}

			return true;
		} catch (e) {
			console.error('Consent removal error:', e);
			return false;
		}
	})()`

	_, err := page.Eval(js)
	return err
}

// SetConsentCookies sets common consent cookies to prevent banners from showing
// Note: This may fail for security reasons (cross-domain restrictions), which is expected
func (c *ConsentRemover) SetConsentCookies(page *rod.Page, domain string) error {
	// Common consent cookies
	cookies := []map[string]interface{}{
		// Generic consent cookies
		{"name": "cookie_consent", "value": "accepted", "domain": domain},
		{"name": "cookies_accepted", "value": "true", "domain": domain},
		{"name": "gdpr_consent", "value": "true", "domain": domain},

		// OneTrust
		{"name": "OptanonAlertBoxClosed", "value": "2024-01-01T00:00:00.000Z", "domain": domain},
		{"name": "OptanonConsent", "value": "isGpcEnabled=0&datestamp=", "domain": domain},

		// CookieBot
		{"name": "CookieConsent", "value": "{stamp:'1',necessary:true,preferences:true,statistics:true,marketing:true}", "domain": domain},

		// Cookie Control
		{"name": "CookieControl", "value": "{necessaryCookies:[],optionalCookies:{}}", "domain": domain},
	}

	js := `(cookies) => {
		try {
			cookies.forEach(cookie => {
				try {
					document.cookie = cookie.name + '=' + cookie.value + '; domain=' + cookie.domain + '; path=/; max-age=31536000';
				} catch (e) {
					// Cookie setting blocked - this is expected for security reasons
				}
			});
			return true;
		} catch (e) {
			return false;
		}
	}`

	_, err := page.Eval(js, cookies)
	return err
}
