package browser

import (
	"os"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// Pool manages a pool of browser instances
type Pool struct {
	browser *rod.Browser
	mu      sync.Mutex
}

// NewPool creates a new browser pool
func NewPool() (*Pool, error) {
	// Launch browser with stealth settings
	l := launcher.New().
		Headless(true).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-features", "IsolateOrigins,site-per-process").
		Set("disable-site-isolation-trials").
		Set("disable-web-security").
		Set("no-first-run").
		Set("no-default-browser-check").
		Set("disable-infobars").
		Set("disable-extensions").
		Set("disable-popup-blocking").
		Set("disable-translate").
		Set("disable-background-networking").
		Set("disable-sync").
		Set("disable-default-apps").
		Set("mute-audio").
		Set("hide-scrollbars").
		Set("no-sandbox").            // Required for Docker
		Set("disable-dev-shm-usage")  // Required for Docker

	// Use custom browser path if specified (for Docker)
	if browserPath := os.Getenv("ROD_BROWSER_PATH"); browserPath != "" {
		l = l.Bin(browserPath)
	}

	url, err := l.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, err
	}

	// Set default timeouts
	browser = browser.Timeout(60 * time.Second)

	return &Pool{browser: browser}, nil
}

// GetPage returns a new stealth page
func (p *Pool) GetPage() (*rod.Page, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create page with stealth mode
	page, err := stealth.Page(p.browser)
	if err != nil {
		return nil, err
	}

	// Set realistic viewport
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             1920,
		Height:            1080,
		DeviceScaleFactor: 1,
		Mobile:            false,
	}); err != nil {
		page.Close()
		return nil, err
	}

	// Set user agent
	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		AcceptLanguage: "en-US,en;q=0.9",
		Platform:       "Win32",
	}); err != nil {
		page.Close()
		return nil, err
	}

	// Add extra evasion via JavaScript
	_, _ = page.Evaluate(rod.Eval(`() => {
		// Override webdriver property
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});

		// Override plugins
		Object.defineProperty(navigator, 'plugins', {
			get: () => [
				{ name: 'Chrome PDF Plugin' },
				{ name: 'Chrome PDF Viewer' },
				{ name: 'Native Client' }
			]
		});

		// Override languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en']
		});

		// Chrome runtime
		window.chrome = {
			runtime: {}
		};

		// Permissions
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);
	}`))

	return page, nil
}

// Close closes the browser
func (p *Pool) Close() error {
	if p.browser != nil {
		return p.browser.Close()
	}
	return nil
}

// NavigateWithRetry navigates to a URL with retry logic
func NavigateWithRetry(page *rod.Page, url string, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := page.Navigate(url)
		if err == nil {
			// Wait for page to be stable
			err = page.WaitStable(2 * time.Second)
			if err == nil {
				return nil
			}
		}
		lastErr = err
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return lastErr
}

// WaitAndClick waits for a selector and clicks it
func WaitAndClick(page *rod.Page, selector string, timeout time.Duration) error {
	el, err := page.Timeout(timeout).Element(selector)
	if err != nil {
		return err
	}
	return el.Click(proto.InputMouseButtonLeft, 1)
}

// GetText extracts text from a selector
func GetText(page *rod.Page, selector string) string {
	el, err := page.Element(selector)
	if err != nil {
		return ""
	}
	text, err := el.Text()
	if err != nil {
		return ""
	}
	return text
}

// GetAttribute gets an attribute value from a selector
func GetAttribute(page *rod.Page, selector, attr string) string {
	el, err := page.Element(selector)
	if err != nil {
		return ""
	}
	val, err := el.Attribute(attr)
	if err != nil || val == nil {
		return ""
	}
	return *val
}

// ScrollToBottom scrolls to the bottom of the page to trigger lazy loading
func ScrollToBottom(page *rod.Page) error {
	_, err := page.Eval(`() => {
		return new Promise((resolve) => {
			let totalHeight = 0;
			const distance = 300;
			const timer = setInterval(() => {
				window.scrollBy(0, distance);
				totalHeight += distance;
				if(totalHeight >= document.body.scrollHeight - window.innerHeight){
					clearInterval(timer);
					resolve(true);
				}
			}, 100);
		});
	}`)
	return err
}

// RandomDelay adds a random human-like delay
func RandomDelay(min, max time.Duration) {
	delay := min + time.Duration(float64(max-min)*0.5) // Simplified random
	time.Sleep(delay)
}
