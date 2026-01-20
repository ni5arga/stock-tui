package chart

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ni5arga/stock-tui/internal/models"
	"github.com/ni5arga/stock-tui/internal/ui/styles"
)

type ChartType int

const (
	ChartLine ChartType = iota
	ChartArea
	ChartCandle
)

var chartTypeNames = []string{"Line", "Area", "Candle"}

type Model struct {
	width      int
	height     int
	symbol     string
	timeRange  models.TimeRange
	chartType  ChartType
	data       []models.Candle
	loading    bool
	err        error
	stale      bool
	retryAfter time.Duration
}

func New() Model {
	return Model{
		timeRange: models.Range24H,
		chartType: ChartLine,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.retryAfter > 0 {
		// Decrement retry timer if we were doing real-time updates,
		// but since we rely on app.go to drive updates, we'll just display what we have.
	}
	return m, nil
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetData(symbol string, tr models.TimeRange, data []models.Candle) {
	m.symbol = symbol
	m.timeRange = tr
	m.data = data
	m.loading = false
	m.err = nil
	m.stale = false
	m.retryAfter = 0
}

func (m *Model) SetStale(retryAfter time.Duration) {
	m.stale = true
	m.retryAfter = retryAfter
	m.loading = false
	m.err = nil
}

func (m *Model) SetLoading(loading bool) { m.loading = loading }

func (m *Model) SetError(err error) {
	m.err = err
	m.loading = false
	m.stale = false
}

func (m *Model) CycleChartType() {
	m.chartType = (m.chartType + 1) % ChartType(len(chartTypeNames))
}

func (m Model) ChartTypeName() string {
	return chartTypeNames[m.chartType]
}

func (m Model) View() string {
	var content string
	switch {
	case m.loading:
		content = lipgloss.Place(m.width-4, m.height-4, lipgloss.Center, lipgloss.Center, "Loading...")
	case m.err != nil:
		content = lipgloss.Place(m.width-4, m.height-4, lipgloss.Center, lipgloss.Center, m.err.Error())
	case len(m.data) == 0:
		content = lipgloss.Place(m.width-4, m.height-4, lipgloss.Center, lipgloss.Center, "No data")
	default:
		content = m.render()
	}

	return styles.ActivePane.Width(m.width).Height(m.height).Render(content)
}

func (m Model) render() string {
	chartH := m.height - 8
	chartW := m.width - 14
	if chartW < 10 || chartH < 4 {
		return "Too small"
	}

	// Get price data
	n := len(m.data)
	closes := make([]float64, n)
	for i, c := range m.data {
		closes[i] = c.Close
	}

	// Find min/max
	minP, maxP := closes[0], closes[0]
	for _, p := range closes {
		if p > 0 && p < minP {
			minP = p
		}
		if p > maxP {
			maxP = p
		}
	}
	spread := maxP - minP
	if spread == 0 {
		spread = maxP * 0.01
	}
	minP -= spread * 0.05
	maxP += spread * 0.05
	spread = maxP - minP

	// Header
	lastP := closes[n-1]
	change := lastP - closes[0]
	pct := change / closes[0] * 100

	up := change >= 0
	trendColor := styles.ColorSuccess
	if !up {
		trendColor = styles.ColorError
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(m.symbol))
	b.WriteString("  ")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.ColorSubtext).Render(string(m.timeRange)))
	b.WriteString("  ")
	b.WriteString(lipgloss.NewStyle().Foreground(trendColor).Bold(true).Render(
		fmt.Sprintf("$%.2f (%+.2f%%)", lastP, pct)))
	b.WriteString("  ")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.ColorSubtext).Render("[" + m.ChartTypeName() + "]"))

	if m.stale {
		warnStyle := lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true)
		b.WriteString("  ")
		b.WriteString(warnStyle.Render(fmt.Sprintf("⚠ RATE LIMITED (Refreshing in %s)", m.retryAfter.Round(time.Second))))
	}
	b.WriteString("\n\n")

	// Build canvas (plain runes, style later per-row)
	canvas := make([][]rune, chartH)
	colors := make([][]bool, chartH) // true = green, false = red
	for i := range canvas {
		canvas[i] = make([]rune, chartW)
		colors[i] = make([]bool, chartW)
		for j := range canvas[i] {
			canvas[i][j] = ' '
			colors[i][j] = true
		}
	}

	toRow := func(price float64) int {
		r := int((maxP - price) / spread * float64(chartH-1))
		if r < 0 {
			r = 0
		}
		if r >= chartH {
			r = chartH - 1
		}
		return r
	}

	// Sample prices to chart width
	step := float64(n) / float64(chartW)

	switch m.chartType {
	case ChartLine:
		prevRow := -1
		for col := 0; col < chartW; col++ {
			idx := int(float64(col) * step)
			if idx >= n {
				idx = n - 1
			}
			row := toRow(closes[idx])
			isUp := idx == 0 || closes[idx] >= closes[max(0, idx-1)]

			if prevRow >= 0 && prevRow != row {
				lo, hi := min(prevRow, row), max(prevRow, row)
				for r := lo; r <= hi; r++ {
					canvas[r][col] = '│'
					colors[r][col] = isUp
				}
			}
			canvas[row][col] = '━'
			colors[row][col] = isUp
			prevRow = row
		}

	case ChartArea:
		for col := 0; col < chartW; col++ {
			idx := int(float64(col) * step)
			if idx >= n {
				idx = n - 1
			}
			row := toRow(closes[idx])
			isUp := idx == 0 || closes[idx] >= closes[max(0, idx-1)]

			for r := row; r < chartH; r++ {
				if r == row {
					canvas[r][col] = '▀'
				} else {
					canvas[r][col] = '░'
				}
				colors[r][col] = isUp
			}
		}

	case ChartCandle:
		// Aggregate candles to fit width
		candlesPerCol := max(1, n/chartW)
		for col := 0; col < chartW; col++ {
			start := col * candlesPerCol
			end := min(start+candlesPerCol, n)
			if start >= n {
				break
			}

			open := m.data[start].Open
			close := m.data[end-1].Close
			high := m.data[start].High
			low := m.data[start].Low
			for i := start; i < end; i++ {
				if m.data[i].High > high {
					high = m.data[i].High
				}
				if m.data[i].Low < low && m.data[i].Low > 0 {
					low = m.data[i].Low
				}
			}

			isUp := close >= open
			rowHigh := toRow(high)
			rowLow := toRow(low)
			rowOpen := toRow(open)
			rowClose := toRow(close)

			bodyTop := min(rowOpen, rowClose)
			bodyBot := max(rowOpen, rowClose)

			// Wick
			for r := rowHigh; r <= rowLow; r++ {
				canvas[r][col] = '│'
				colors[r][col] = isUp
			}
			// Body
			for r := bodyTop; r <= bodyBot; r++ {
				if isUp {
					canvas[r][col] = '█'
				} else {
					canvas[r][col] = '▓'
				}
				colors[r][col] = isUp
			}
		}
	}

	// Render canvas with colors
	greenS := lipgloss.NewStyle().Foreground(styles.ColorSuccess)
	redS := lipgloss.NewStyle().Foreground(styles.ColorError)
	dimS := lipgloss.NewStyle().Foreground(styles.ColorSubtext)

	for row := 0; row < chartH; row++ {
		// Y-axis label
		var label string
		switch row {
		case 0:
			label = fmt.Sprintf("%8.2f ", maxP)
		case chartH - 1:
			label = fmt.Sprintf("%8.2f ", minP)
		case chartH / 2:
			label = fmt.Sprintf("%8.2f ", (maxP+minP)/2)
		default:
			label = "         "
		}
		b.WriteString(dimS.Render(label))

		// Chart row - batch same-color runs for cleaner output
		var rowStr strings.Builder
		for col := 0; col < chartW; col++ {
			ch := canvas[row][col]
			if colors[row][col] {
				rowStr.WriteString(greenS.Render(string(ch)))
			} else {
				rowStr.WriteString(redS.Render(string(ch)))
			}
		}
		b.WriteString(rowStr.String())
		b.WriteString("\n")
	}

	// Sparkline
	b.WriteString("\n")
	b.WriteString(m.sparkline(closes, chartW))

	return b.String()
}

func (m Model) sparkline(prices []float64, width int) string {
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	n := len(prices)
	if n == 0 {
		return ""
	}

	minP, maxP := prices[0], prices[0]
	for _, p := range prices {
		if p < minP {
			minP = p
		}
		if p > maxP {
			maxP = p
		}
	}
	rng := maxP - minP
	if rng == 0 {
		rng = 1
	}

	step := float64(n) / float64(width)
	greenS := lipgloss.NewStyle().Foreground(styles.ColorSuccess)
	redS := lipgloss.NewStyle().Foreground(styles.ColorError)

	var out strings.Builder
	out.WriteString(lipgloss.NewStyle().Foreground(styles.ColorSubtext).Render("   Trend "))

	prev := prices[0]
	for i := 0; i < width; i++ {
		idx := int(float64(i) * step)
		if idx >= n {
			idx = n - 1
		}
		p := prices[idx]
		norm := (p - minP) / rng
		bi := int(norm * float64(len(blocks)-1))
		bi = max(0, min(bi, len(blocks)-1))

		if p >= prev {
			out.WriteString(greenS.Render(string(blocks[bi])))
		} else {
			out.WriteString(redS.Render(string(blocks[bi])))
		}
		prev = p
	}

	return out.String()
}
