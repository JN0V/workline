// Package rates knows the currencies with a rate.
package rates

// Known says whether a currency has a rate.
func Known(code string) bool {
	return code == "EUR" || code == "USD" // rates body
}
