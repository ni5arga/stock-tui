package app

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ni5arga/stock-tui/internal/data"
	"github.com/ni5arga/stock-tui/internal/models"
	"github.com/ni5arga/stock-tui/internal/ui/chart"
	"github.com/ni5arga/stock-tui/internal/ui/footer"
	"github.com/ni5arga/stock-tui/internal/ui/help"
	"github.com/ni5arga/stock-tui/internal/ui/watchlist"
)

type AppModel struct {
	cfg      *models.AppConfig
	provider data.Provider

	watchlist watchlist.Model
	chart     chart.Model
	footer    footer.Model
	help      help.Model

	width  int
	height int

	timeRange     models.TimeRange
	refreshTicker *time.Ticker
	lastQuotes    []models.Quote
	lastHistory   map[string][]models.Candle
	err           error
}

type tickMsg time.Time

type quotesMsg struct {
	quotes []models.Quote
	err    error
}

type historyMsg struct {
	symbol string
	tr     models.TimeRange
	data   []models.Candle
	err    error
}

type retryHistoryMsg struct {
	symbol string
	tr     models.TimeRange
}

func New(cfg *models.AppConfig) (*AppModel, error) {
	prov, _ := data.NewProvider(cfg.Provider)

	tr := models.Range24H
	switch cfg.DefaultRange {
	case "1H":
		tr = models.Range1H
	case "7D":
		tr = models.Range7D
	case "30D":
		tr = models.Range30D
	}

	return &AppModel{
		cfg:         cfg,
		provider:    prov,
		watchlist:   watchlist.New(cfg.Symbols),
		chart:       chart.New(),
		footer:      footer.New(prov.Name()),
		help:        help.New(),
		timeRange:   tr,
		lastHistory: make(map[string][]models.Candle),
	}, nil
}

func (m *AppModel) Init() tea.Cmd {
	m.refreshTicker = time.NewTicker(m.cfg.RefreshInterval)

	return tea.Batch(
		tea.EnterAltScreen,
		m.fetchQuotes(),
		m.fetchAllHistory(),
		m.waitForTick(),
	)
}

func (m *AppModel) waitForTick() tea.Cmd {
	return func() tea.Msg {
		t := <-m.refreshTicker.C
		return tickMsg(t)
	}
}

func (m *AppModel) fetchQuotes() tea.Cmd {
	return func() tea.Msg {
		quotes, err := m.provider.GetQuotes(m.cfg.Symbols)
		return quotesMsg{quotes: quotes, err: err}
	}
}

func (m *AppModel) fetchHistory(symbol string, tr models.TimeRange) tea.Cmd {
	return func() tea.Msg {
		h, err := m.provider.GetHistory(symbol, tr)
		return historyMsg{symbol: symbol, tr: tr, data: h, err: err}
	}
}

func (m *AppModel) fetchAllHistory() tea.Cmd {
	// Batch fetch history for all symbols
	cmds := make([]tea.Cmd, 0, len(m.cfg.Symbols))
	for _, sym := range m.cfg.Symbols {
		s := sym // capture
		cmds = append(cmds, func() tea.Msg {
			h, err := m.provider.GetHistory(s, m.timeRange)
			return historyMsg{symbol: s, tr: m.timeRange, data: h, err: err}
		})
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.help.Visible() {
		m.help, cmd = m.help.Update(msg)
		cmds = append(cmds, cmd)
		if !m.help.Visible() {
			return m, tea.Batch(cmds...)
		}
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() != "?" && msg.String() != "esc" {
				return m, tea.Batch(cmds...)
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.help.SetSize(msg.Width, msg.Height)
		}
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Footer click (Cycle time range)
			if msg.Y == m.height-1 {
				m.cycleTimeRange()
				return m, m.refreshCurrentChart()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		footerHeight := 1
		mainHeight := m.height - footerHeight

		wlWidth := int(float64(m.width) * 0.28)
		if wlWidth < 30 {
			wlWidth = 30
		}
		if wlWidth > 45 {
			wlWidth = 45
		}
		chartWidth := m.width - wlWidth

		m.watchlist.SetSize(wlWidth, mainHeight)
		m.chart.SetSize(chartWidth, mainHeight)
		m.footer.SetSize(m.width, footerHeight)
		m.help.SetSize(m.width, m.height)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "?":
			m.help.Toggle()
			return m, nil

		case "tab":
			m.cycleTimeRange()
			return m, m.loadCurrentChart()

		case "1":
			m.setTimeRange(models.Range1H)
			return m, m.loadCurrentChart()
		case "2":
			m.setTimeRange(models.Range24H)
			return m, m.loadCurrentChart()
		case "3":
			m.setTimeRange(models.Range7D)
			return m, m.loadCurrentChart()
		case "4":
			m.setTimeRange(models.Range30D)
			return m, m.loadCurrentChart()

		case "r":
			return m, tea.Batch(m.fetchQuotes(), m.refreshCurrentChart())

		case "c":
			m.chart.CycleChartType()
			return m, nil
		}

	case tickMsg:
		cmds = append(cmds, m.fetchQuotes(), m.waitForTick())

	case quotesMsg:
		if msg.err != nil {
			m.err = msg.err
			m.footer.SetStatus(time.Now(), false, msg.err)
		} else {
			m.lastQuotes = msg.quotes
			m.watchlist.UpdateQuotes(msg.quotes)
			m.footer.SetStatus(time.Now(), true, nil)
			m.err = nil

			sel := m.watchlist.SelectedSymbol()
			if sel != "" {
				cacheKey := sel + "|" + string(m.timeRange)
				if _, ok := m.lastHistory[cacheKey]; !ok {
					m.chart.SetLoading(true)
					cmds = append(cmds, m.fetchHistory(sel, m.timeRange))
				}
			}
		}

	case retryHistoryMsg:
		if m.watchlist.SelectedSymbol() == msg.symbol && m.timeRange == msg.tr {
			m.chart.SetLoading(true)
		}
		cmds = append(cmds, m.fetchHistory(msg.symbol, msg.tr))

	case historyMsg:
		if msg.err != nil {
			var rateLimitErr *data.RateLimitError
			if errors.As(msg.err, &rateLimitErr) {
				cacheKey := msg.symbol + "|" + string(msg.tr)
				if cached, ok := m.lastHistory[cacheKey]; ok {
					if m.watchlist.SelectedSymbol() == msg.symbol && m.timeRange == msg.tr {
						m.chart.SetData(msg.symbol, msg.tr, cached)
						m.chart.SetStale(rateLimitErr.RetryAfter)
					}
				} else {
					m.chart.SetError(msg.err)
				}

				// Auto-retry after delay
				cmds = append(cmds, tea.Tick(rateLimitErr.RetryAfter, func(t time.Time) tea.Msg {
					return retryHistoryMsg{symbol: msg.symbol, tr: msg.tr}
				}))
				return m, tea.Batch(cmds...)
			}
			m.chart.SetError(msg.err)
		} else {
			cacheKey := msg.symbol + "|" + string(msg.tr)
			m.lastHistory[cacheKey] = msg.data
			if m.watchlist.SelectedSymbol() == msg.symbol && m.timeRange == msg.tr {
				m.chart.SetData(msg.symbol, msg.tr, msg.data)
			}
			// Update watchlist with % change from history (start to end)
			if len(msg.data) > 1 {
				startPrice := msg.data[0].Close
				endPrice := msg.data[len(msg.data)-1].Close
				m.watchlist.UpdatePriceChange(msg.symbol, endPrice, startPrice)
			}
		}
	}

	oldSel := m.watchlist.SelectedSymbol()
	m.watchlist, cmd = m.watchlist.Update(msg)
	cmds = append(cmds, cmd)

	newSel := m.watchlist.SelectedSymbol()
	if oldSel != newSel && newSel != "" {
		cacheKey := newSel + "|" + string(m.timeRange)
		if cached, ok := m.lastHistory[cacheKey]; ok {
			m.chart.SetData(newSel, m.timeRange, cached)
		} else {
			m.chart.SetLoading(true)
			cmds = append(cmds, m.fetchHistory(newSel, m.timeRange))
		}
	}

	m.chart, cmd = m.chart.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *AppModel) cycleTimeRange() {
	ranges := []models.TimeRange{models.Range1H, models.Range24H, models.Range7D, models.Range30D}
	for i, tr := range ranges {
		if tr == m.timeRange {
			m.timeRange = ranges[(i+1)%len(ranges)]
			break
		}
	}
	m.footer.SetTimeRange(m.timeRange)
}

func (m *AppModel) setTimeRange(tr models.TimeRange) {
	if m.timeRange == tr {
		return
	}
	m.timeRange = tr
	m.footer.SetTimeRange(m.timeRange)
}

func (m *AppModel) refreshCurrentChart() tea.Cmd {
	sel := m.watchlist.SelectedSymbol()
	if sel == "" {
		return nil
	}
	m.chart.SetLoading(true)
	return m.fetchHistory(sel, m.timeRange)
}

func (m *AppModel) loadCurrentChart() tea.Cmd {
	sel := m.watchlist.SelectedSymbol()
	if sel == "" {
		return nil
	}
	cacheKey := sel + "|" + string(m.timeRange)
	if cached, ok := m.lastHistory[cacheKey]; ok {
		m.chart.SetData(sel, m.timeRange, cached)
		return nil
	}
	m.chart.SetLoading(true)
	return m.fetchHistory(sel, m.timeRange)
}

func (m *AppModel) View() string {
	main := lipgloss.JoinHorizontal(lipgloss.Top, m.watchlist.View(), m.chart.View())
	base := lipgloss.JoinVertical(lipgloss.Left, main, m.footer.View())

	if m.help.Visible() {
		helpView := m.help.View()
		return overlayModal(base, helpView, m.width, m.height)
	}

	return base
}

func (m *AppModel) Close() {
	if m.refreshTicker != nil {
		m.refreshTicker.Stop()
	}
}

func overlayModal(base, modal string, w, h int) string {
	if modal == "" {
		return base
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
}
