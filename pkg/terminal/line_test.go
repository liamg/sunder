package terminal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLine(t *testing.T) {

	line := newLine()
	line.cells = []Cell{
		{r: MeasuredRune{Rune: 'h'}},
		{r: MeasuredRune{Rune: 'e'}},
		{r: MeasuredRune{Rune: 'l'}},
		{r: MeasuredRune{Rune: 'l'}},
		{r: MeasuredRune{Rune: 'o'}},
	}

	assert.Equal(t, "hello", line.String())
	assert.False(t, line.wrapped)

	line.setWrapped(true)
	assert.True(t, line.wrapped)

	line.setWrapped(false)
	assert.False(t, line.wrapped)

}
