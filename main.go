package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

const cookieFile = "cookies.json"

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, proceeding without it.")
	}

	email := os.Getenv("EMAIL")
	password := os.Getenv("PASSWORD")

	if email == "" || password == "" {
		log.Fatal("EMAIL or PASSWORD is not set in the environment variables")
	}

	// Create a context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set timeout for the overall process
	ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Start the browser and restore cookies if available
	if err := restoreCookies(ctx); err != nil {
		log.Printf("Failed to restore cookies: %v", err)
	}

	// Check if we're logged in; if not, perform login
	if err := ensureLoggedIn(ctx, email, password); err != nil {
		log.Fatalf("Failed to log in: %v", err)
	}

	// Save cookies for future runs
	if err := saveCookies(ctx); err != nil {
		log.Printf("Failed to save cookies: %v", err)
	}

	// Proceed with the main task
	startURL := "https://typst.app/team/aKj7S1kHEc96JAgoh1C5Ri"
	var links []string

	// Step 1: Navigate to the start URL and extract links
	err := chromedp.Run(ctx,
		chromedp.Navigate(startURL),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('main > *:nth-child(2) > * a')).map(a => a.href)`, &links),
	)
	if err != nil {
		log.Fatalf("Failed to get links: %v", err)
	}

	log.Printf("Found %d links", len(links))

	// Step 2: Process each link
	for _, link := range links {
		log.Printf("Processing link: %s", link)
		if err := processLink(ctx, link); err != nil {
			log.Printf("Error processing link %s: %v", link, err)
		}
	}
}

func ensureLoggedIn(ctx context.Context, username, password string) error {
	// Check if already logged in
	var loggedIn bool
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://typst.app/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.body.textContent.includes("Sign out")`, &loggedIn),
	)
	if err != nil {
		return err
	}

	if loggedIn {
		log.Println("Already logged in")
		return nil
	}

	// Perform login
	log.Println("Logging in...")
	err = chromedp.Run(ctx,
		chromedp.Navigate("https://typst.app/signin"),
		chromedp.WaitVisible(`#email`, chromedp.ByID),
		chromedp.SendKeys(`#email`, username, chromedp.ByID),
		chromedp.SendKeys(`#password`, password, chromedp.ByID),
		chromedp.Click(`input[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // Wait for login to complete
	)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	log.Println("Login successful")
	return nil
}

func saveCookies(ctx context.Context) error {
	var cookies []*chromedp.Cookie
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = chromedp.Cookies(ctx)
			return err
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	file, err := os.Create(cookieFile)
	if err != nil {
		return fmt.Errorf("failed to create cookie file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cookies); err != nil {
		return fmt.Errorf("failed to encode cookies: %w", err)
	}

	log.Printf("Cookies saved to %s", cookieFile)
	return nil
}

func restoreCookies(ctx context.Context) error {
	file, err := os.Open(cookieFile)
	if os.IsNotExist(err) {
		log.Println("No cookie file found, skipping cookie restoration.")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to open cookie file: %w", err)
	}
	defer file.Close()

	var cookies []*chromedp.Cookie
	if err := json.NewDecoder(file).Decode(&cookies); err != nil {
		return fmt.Errorf("failed to decode cookies: %w", err)
	}

	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, cookie := range cookies {
				if err := chromedp.SetCookie(cookie).Do(ctx); err != nil {
					log.Printf("Failed to set cookie %s: %v", cookie.Name, err)
				}
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to restore cookies: %w", err)
	}

	log.Println("Cookies restored")
	return nil
}

func processLink(ctx context.Context, link string) error {
	var zipFileName string

	// Navigate to the link, click the "File" button and "Backup project" parent
	err := chromedp.Run(ctx,
		chromedp.Navigate(link),
		chromedp.Sleep(2*time.Second), // Wait for the page to load

		// Click the <button> with textContent "File"
		chromedp.Click(`//button[contains(text(), 'File')]`, chromedp.NodeVisible),

		// Click the parent of <span> with textContent "Backup project"
		chromedp.Click(`//span[contains(text(), 'Backup project')]/..`, chromedp.NodeVisible),

		// Wait for the download (depends on your Chrome setup; use appropriate handling)
		chromedp.ActionFunc(func(ctx context.Context) error {
			zipFileName = fmt.Sprintf("backup_%d.zip", time.Now().Unix())
			log.Printf("Simulated saving: %s", zipFileName)
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to process link %s: %w", link, err)
	}

	log.Printf("Processed link: %s, saved file as: %s", link, zipFileName)
	return nil
}

