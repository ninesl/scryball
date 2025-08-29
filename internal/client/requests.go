package client

import (
	"fmt"
	"net/url"
)

// QueryForCards searches the Scryfall API using a query string and returns ALL matching cards
// This function uses the /cards/search endpoint with the provided query
// Handles pagination to retrieve ALL cards across all pages, not just the first page
// Returns an array of Cards or an error if the request fails
func (c *Client) QueryForCards(scryfallQuery string) ([]Card, error) {
	var allCards []Card

	// Get first page
	var list List
	err := c.makeRequest("/cards/search?q="+url.QueryEscape(scryfallQuery), &list)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards with query '%s': %w", scryfallQuery, err)
	}

	// Add first page results
	allCards = append(allCards, list.Data...)

	// Follow pagination to get all pages
	for list.HasMore && list.NextPage != nil {
		// Extract the path and query from the next page URL
		nextEndpoint := list.NextPage.Path
		if list.NextPage.RawQuery != "" {
			nextEndpoint += "?" + list.NextPage.RawQuery
		}

		// Make request for next page
		err = c.makeRequest(nextEndpoint, &list)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch next page: %w", err)
		}

		// Add this page's results
		allCards = append(allCards, list.Data...)
	}

	return allCards, nil
}

// QueryForCard searches the Scryfall API for a specific card by exact name
// This function uses the /cards/named endpoint to find cards by exact name match
// Returns a single Card or an error if not found or request fails
func (c *Client) QueryForSpecificCard(cardName string) (*Card, error) {
	var card Card
	// Use the /cards/named endpoint with exact parameter for precise matching
	endpoint := "/cards/named?exact=" + url.QueryEscape(cardName)
	err := c.makeRequest(endpoint, &card)
	if err != nil {
		return nil, fmt.Errorf("failed to find card with name '%s': %w", cardName, err)
	}
	return &card, nil
}
