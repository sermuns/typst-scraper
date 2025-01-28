package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/runtime"
	"log"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
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

	// Create a context with specific options
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()

	// Create a browser context
	parentCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout for the overall process
	parentCtx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
	defer cancel()

	// Check if we're logged in; if not, perform login
	if err := login(parentCtx, email, password); err != nil {
		log.Fatalf("Failed to log in: %v", err)
	}

	// Proceed with the main task
	startURL := "https://typst.app/team/aKj7S1kHEc96JAgoh1C5Ri"
	var links []string

	// Step 1: Navigate to the start URL and extract links
	err := chromedp.Run(parentCtx,
		chromedp.Navigate(startURL),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('main > *:nth-child(2) > * a')).map(a => a.href)`, &links),
	)
	if err != nil {
		log.Fatalf("Failed to get links: %v", err)
	}

	log.Printf("Found %d links", len(links))

	// Step 2: Process each link in parallel
	var wg sync.WaitGroup
	for _, link := range links {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()

			// Create a new tab (context) for each link
			tabCtx, cancel := chromedp.NewContext(parentCtx)
			defer cancel()

			if err := processLink(tabCtx, link); err != nil {
				log.Printf("Error processing link %s: %v", link, err)
			}
		}(link)
	}

	// Wait for all links to be processed
	wg.Wait()
	log.Println("All links processed.")
}

func login(ctx context.Context, username, password string) error {
	log.Println("Logging in...")
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://typst.app/signin"),
		chromedp.WaitVisible(`#email`, chromedp.ByID),
		chromedp.SendKeys(`#email`, username, chromedp.ByID),
		chromedp.SendKeys(`#password`, password, chromedp.ByID),
		chromedp.Click(`input[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Wait for login to complete
	)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	log.Println("Login successful")
	return nil
}

func processLink(ctx context.Context, link string) error {
	var zipFileName string

	// Navigate to the link, click the "File" button and "Backup project" parent
	err := chromedp.Run(ctx,
		chromedp.Navigate(link),
		chromedp.Sleep(10*time.Second), // Wait for the page to load

		// Select the "File" button node
		chromedp.QueryAfter(`span`,
			func(ctx context.Context, eci runtime.ExecutionContextID, nodes ...*cdp.Node) error {
				for _, node := range nodes {
					// find one with correct innertext, click it.
					var innerText string
					if err := chromedp.Text(node, &innerText).Do(ctx); err != nil {
						return err
					}

					// Check if the text matches "File"
					if innerText == "File" {
						// Perform the mouse click on the node
						return chromedp.MouseClickNode(node).Do(ctx)
					}
				}
				return errors.New("Couldnt find File button")
			}, chromedp.ByID),

		// // Select the parent of <span> with textContent "Backup project" node
		// chromedp.Nodes(`span[text()='Backup project']/..`, &nodes), // Select the element
		// chromedp.MouseClickNode(nodes[0]),                          // Click the node

		// Wait for the download (depends on your Chrome setup; use appropriate handling)
		chromedp.ActionFunc(func(ctx context.Context) error {
			zipFileName = fmt.Sprintf("backup_%d.zip", time.Now().Unix())
			log.Printf("Download initiated: %s", zipFileName)
			// File should be downloaded automatically by the browser to the specified directory
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to process link %s: %w", link, err)
	}

	return nil
}
