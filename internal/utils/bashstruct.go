package utils

import "time"

// BashStruct describes model of the parsed bash.im quotes.
type BashStruct struct {
	Date   time.Time `json:"-"`
	Number string    `json:"-"`
	Text   string    `json:"text"`
}

// GetText returns text of quote.
func (b *BashStruct) GetText() string {
	return b.Text
}
