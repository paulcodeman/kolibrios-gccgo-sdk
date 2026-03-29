package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"kos"
	"ui"
	"ui/elements"
)

const (
	appTitle                 = "VK Messenger"
	authWindowWidth          = 520
	authWindowHeight         = 420
	windowWidth              = 1120
	windowHeight             = 760
	sidebarDefaultWidth      = 320
	sidebarCompactWidth      = 280
	layoutGap                = 12
	windowPadding            = 12
	shellHeaderReservedHeight = 116
	contentMinimumHeight     = 260
	conversationHeaderHeight = 48
	messageReservedHeight    = 212
	conversationPageSize     = 30
	historyPageSize          = 60
	defaultAPIBase           = "https://api.vk.ru/method/"
	defaultAPIVersion        = "5.199"
	localCABundlePath        = "assets/ca-bundle.pem"
	defaultFontPath          = "assets/OpenSans-Regular.ttf"
	monoFontPath             = "assets/RobotoMono-Regular.ttf"
	settingsConfigPath       = "assets/settings.json"
	settingsExamplePath      = "assets/settings.example.json"
)

const (
	colorWindowBG       = kos.Color(0xEEF3F8)
	colorPanelBG        = kos.Color(0xFFFFFF)
	colorPanelBorder    = kos.Color(0xD7E2EC)
	colorVKBlue         = kos.Color(0x2787F5)
	colorVKBlueDark     = kos.Color(0x1059B8)
	colorSidebarBG      = kos.Color(0xF8FAFC)
	colorSoftBlue       = kos.Color(0xE8F2FF)
	colorSoftIncoming   = kos.Color(0xF3F6F9)
	colorMeta           = kos.Color(0x6A7A8C)
	colorText           = kos.Color(0x1B2B38)
	colorSuccess        = kos.Color(0x138A36)
	colorDanger         = kos.Color(0xB3261E)
	colorComposerBG     = kos.Color(0xFBFCFE)
	colorSelectedCard   = kos.Color(0xEEF5FF)
	colorSelectedBorder = kos.Color(0x7FB1F5)
)

// TYPES

type appSession struct {
	AccessToken string `json:"access_token"`
	APIBase     string `json:"api_base"`
	APIVersion  string `json:"api_version"`
}

type vkClient struct {
	baseURL     string
	apiVersion  string
	accessToken string
	httpClient  *http.Client
}

type vkAPIError struct {
	Code    int    `json:"error_code"`
	Message string `json:"error_msg"`
}

func (err *vkAPIError) Error() string {
	if err == nil {
		return ""
	}
	if err.Code == 0 {
		return strings.TrimSpace(err.Message)
	}
	return "VK API " + strconv.Itoa(err.Code) + ": " + strings.TrimSpace(err.Message)
}

type vkErrorEnvelope struct {
	Error *vkAPIError `json:"error"`
}

type vkUser struct {
	ID         int64       `json:"id"`
	FirstName  string      `json:"first_name"`
	LastName   string      `json:"last_name"`
	ScreenName string      `json:"screen_name"`
	Photo100   string      `json:"photo_100"`
	Online     int         `json:"online"`
	LastSeen   *vkLastSeen `json:"last_seen"`
}

type vkLastSeen struct {
	Time int64 `json:"time"`
}

type vkGroup struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
	Photo100   string `json:"photo_100"`
}

type vkUserGetResponse struct {
	Response []vkUser `json:"response"`
}

type vkPeer struct {
	ID      int64  `json:"id"`
	LocalID int    `json:"local_id"`
	Type    string `json:"type"`
}

type vkConversationCanWrite struct {
	Allowed bool  `json:"allowed"`
	Reason  int   `json:"reason"`
	Until   int64 `json:"until"`
}

type vkChatSettings struct {
	Title        string `json:"title"`
	MembersCount int    `json:"members_count"`
}

type vkConversation struct {
	Peer          vkPeer                  `json:"peer"`
	UnreadCount   int                     `json:"unread_count"`
	LastMessageID int                     `json:"last_message_id"`
	InRead        int                     `json:"in_read"`
	OutRead       int                     `json:"out_read"`
	CanWrite      *vkConversationCanWrite `json:"can_write"`
	ChatSettings  *vkChatSettings         `json:"chat_settings"`
}

type vkMessageAction struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	MemberID int64  `json:"member_id"`
	Message  string `json:"message"`
}

type vkMessageAttachment struct {
	Type string `json:"type"`
}

type vkForeignMessage struct {
	FromID      int64                 `json:"from_id"`
	Text        string                `json:"text"`
	Attachments []vkMessageAttachment `json:"attachments"`
	Action      *vkMessageAction      `json:"action"`
}

type vkMessage struct {
	ID                    int                   `json:"id"`
	ConversationMessageID int                   `json:"conversation_message_id"`
	Date                  int64                 `json:"date"`
	FromID                int64                 `json:"from_id"`
	PeerID                int64                 `json:"peer_id"`
	Text                  string                `json:"text"`
	Out                   int                   `json:"out"`
	UpdateTime            int64                 `json:"update_time"`
	Attachments           []vkMessageAttachment `json:"attachments"`
	Action                *vkMessageAction      `json:"action"`
	ReplyMessage          *vkForeignMessage     `json:"reply_message"`
	FwdMessages           []vkForeignMessage    `json:"fwd_messages"`
}

type vkConversationWithMessage struct {
	Conversation vkConversation `json:"conversation"`
	LastMessage  vkMessage      `json:"last_message"`
}

type vkGetConversationsResponse struct {
	Response struct {
		Count       int                         `json:"count"`
		UnreadCount int                         `json:"unread_count"`
		Items       []vkConversationWithMessage `json:"items"`
		Profiles    []vkUser                    `json:"profiles"`
		Groups      []vkGroup                   `json:"groups"`
	} `json:"response"`
}

type vkGetHistoryResponse struct {
	Response struct {
		Count    int         `json:"count"`
		Items    []vkMessage `json:"items"`
		Profiles []vkUser    `json:"profiles"`
		Groups   []vkGroup   `json:"groups"`
	} `json:"response"`
}

type vkSendResponseItem struct {
	PeerID                int64 `json:"peer_id"`
	MessageID             int   `json:"message_id"`
	ConversationMessageID int   `json:"conversation_message_id"`
	Error                 *struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"error"`
}

type vkSendEnvelope struct {
	Response json.RawMessage `json:"response"`
}

// APP

type App struct {
	window      *ui.Window
	httpClient  *http.Client
	session     appSession
	started     bool
	authorized  bool
	currentUser *vkUser

	authRoot    *ui.Element
	authCard    *ui.Element
	statusLine        *ui.Element
	profileLine       *ui.Element
	shellRoot         *ui.Element
	conversationHead  *ui.Element
	conversationMeta  *ui.Element
	sendButton        *ui.Element
	composeInput      *ui.Element
	sidebarPanel      *ui.Element
	mainPanel         *ui.Element
	conversationsView *ui.DocumentView
	messagesView      *ui.DocumentView

	selectedPeerID     int64
	conversations      []vkConversationWithMessage
	conversationUsers  map[int64]vkUser
	conversationGroups map[int64]vkGroup
	historyMessages    []vkMessage
	historyUsers       map[int64]vkUser
	historyGroups      map[int64]vkGroup
}

func NewApp() *App {
	session, sessionErr := loadSession()
	caBundlePath, _ := configureLocalCABundle()
	rootCAs, caErr := loadRootPool(caBundlePath)
	app := &App{
		httpClient:         newHTTPClient(rootCAs),
		session:            session,
		conversationUsers:  map[int64]vkUser{},
		conversationGroups: map[int64]vkGroup{},
		historyUsers:       map[int64]vkUser{},
		historyGroups:      map[int64]vkGroup{},
	}
	app.buildUI()
	if strings.TrimSpace(session.AccessToken) != "" {
		app.setStatus("В "+settingsConfigPath+" найден access_token. Нажмите кнопку, чтобы загрузить диалоги.", colorVKBlueDark)
	} else {
		app.setStatus("Добавьте access_token в "+settingsConfigPath+", затем перезапустите приложение или нажмите Прочитать settings.json.", colorMeta)
	}
	if sessionErr != nil {
		app.setStatus("Не удалось прочитать "+settingsConfigPath+": "+sessionErr.Error(), colorDanger)
	} else if caErr != nil {
		app.setStatus("TLS roots: "+caErr.Error(), colorDanger)
	}
	app.showAuthScreen()
	return app
}

func (app *App) Run() {
	if strings.TrimSpace(app.session.AccessToken) != "" {
		_ = app.connectAndRefresh(false)
	}
	app.started = true
	app.window.Start()
}

// UI

func (app *App) buildUI() {
	windowX, windowY, windowW, windowH := desktopWindowBounds()
	window := ui.NewWindowDefault()
	window.SetTitle(appTitle)
	window.UpdateStyle(func(style *ui.Style) {
		style.SetLeft(windowX)
		style.SetTop(windowY)
		style.SetWidth(windowW)
		style.SetHeight(windowH)
		style.SetOverflow(ui.OverflowAuto)
		style.SetBackground(colorWindowBG)
	})
	window.SetBounds(windowX, windowY, windowW, windowH)
	window.OnResize = app.handleResize
	app.window = window

	root := ui.CreateBox()
	applyStyle(root, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(windowPadding)
		style.SetBackground(colorWindowBG)
	})

	header := ui.CreateBox()
	applyStyle(header, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12, 14)
		style.SetMargin(0, 0, 10, 0)
		style.SetBorderRadius(16)
		style.SetGradient(ui.Gradient{
			From:      colorVKBlueDark,
			To:        colorVKBlue,
			Direction: ui.GradientHorizontal,
		})
		style.SetShadow(ui.Shadow{OffsetX: 0, OffsetY: 2, Blur: 5, Color: ui.Black, Alpha: 55})
	})

	title := elements.Label("VK Messenger")
	applyStyle(title, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.White)
		style.SetFontSize(22)
		style.SetMargin(0, 0, 4, 0)
	})

	subtitle := elements.Label("Нативный клиент на stdlib/ui и VK API. Авторизация берётся только из settings.json, затем окно разворачивается в список диалогов.")
	applyStyle(subtitle, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorSoftBlue)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})

	header.Append(title)
	header.Append(subtitle)

	infoPanel := ui.CreateBox()
	applyStyle(infoPanel, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(10, 12)
		style.SetMargin(0, 0, 12, 0)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(12)
		style.SetBackground(colorPanelBG)
	})

	profileLine := elements.Label("Профиль: не авторизован")
	applyStyle(profileLine, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorText)
		style.SetFontSize(12)
		style.SetMargin(0, 0, 4, 0)
	})
	app.profileLine = profileLine

	statusLine := elements.Label("Статус: ожидание")
	applyStyle(statusLine, true, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorMeta)
		style.SetFontSize(12)
		style.SetPadding(8, 10)
		style.SetBackground(colorSidebarBG)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(8)
	})
	app.statusLine = statusLine

	infoPanel.Append(profileLine)
	infoPanel.Append(statusLine)

	authRoot := ui.CreateBox()
	applyStyle(authRoot, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(0)
		style.SetMargin(0)
	})
	app.authRoot = authRoot

	authCard := ui.CreateBox()
	applyStyle(authCard, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(16, 18)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(16)
		style.SetBackground(colorPanelBG)
		style.SetShadow(ui.Shadow{OffsetX: 0, OffsetY: 2, Blur: 6, Color: ui.Black, Alpha: 36})
	})
	app.authCard = authCard

	authTitle := elements.Label("Настройки")
	applyStyle(authTitle, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorVKBlueDark)
		style.SetFontSize(18)
		style.SetMargin(0, 0, 6, 0)
	})

	authHint := elements.Label("Приложение читает только "+settingsConfigPath+". Укажите там готовый access_token VK со scope messages. Шаблон лежит в "+settingsExamplePath+".")
	applyStyle(authHint, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorMeta)
		style.SetFontSize(11)
		style.SetLineHeight(15)
		style.SetMargin(0, 0, 10, 0)
	})

	settingsFields := elements.Label("Ожидаемые поля:\n- access_token\n- api_base\n- api_version\n\nЕсли token уже сохранён, нажмите кнопку ниже.")
	applyStyle(settingsFields, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorText)
		style.SetFontSize(12)
		style.SetLineHeight(18)
		style.SetWhiteSpace(ui.WhiteSpacePreWrap)
		style.SetMargin(0, 0, 10, 0)
	})

	authButtons := ui.CreateBox()
	applyStyle(authButtons, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 0, 0)
	})

	connectButton := elements.Button("Прочитать settings.json")
	styleActionButton(connectButton, colorVKBlue, ui.White, colorVKBlueDark)
	connectButton.OnClick = func() {
		_ = app.connectAndRefresh(true)
	}
	authButtons.Append(connectButton)

	authCard.Append(authTitle)
	authCard.Append(authHint)
	authCard.Append(settingsFields)
	authCard.Append(authButtons)
	authRoot.Append(authCard)

	shellRoot := ui.CreateBox()
	applyStyle(shellRoot, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayNone)
		style.SetPadding(0)
		style.SetMargin(0)
	})
	app.shellRoot = shellRoot

	toolbar := ui.CreateBox()
	applyStyle(toolbar, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12, 14)
		style.SetMargin(0, 0, 12, 0)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(14)
		style.SetBackground(colorPanelBG)
	})

	reloadSettingsButton := elements.Button("Перечитать settings.json")
	styleActionButton(reloadSettingsButton, colorVKBlue, ui.White, colorVKBlueDark)
	reloadSettingsButton.OnClick = func() {
		_ = app.connectAndRefresh(true)
	}

	refreshConversationsButton := elements.Button("Диалоги")
	styleActionButton(refreshConversationsButton, colorSoftBlue, colorVKBlueDark, colorSelectedBorder)
	refreshConversationsButton.OnClick = func() {
		_ = app.refreshConversations(true, true)
	}

	refreshHistoryButton := elements.Button("История")
	styleActionButton(refreshHistoryButton, colorSoftIncoming, colorText, colorPanelBorder)
	refreshHistoryButton.OnClick = func() {
		_ = app.loadHistoryForSelection(true)
	}

	toolbar.Append(reloadSettingsButton)
	toolbar.Append(refreshConversationsButton)
	toolbar.Append(refreshHistoryButton)

	content := ui.CreateBox()
	applyStyle(content, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
	})

	sidebar := ui.CreateBox()
	applyStyle(sidebar, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, layoutGap, 0, 0)
		style.SetPadding(10)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(14)
		style.SetBackground(colorPanelBG)
	})
	app.sidebarPanel = sidebar

	sidebarTitle := elements.Label("Диалоги")
	applyStyle(sidebarTitle, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorVKBlueDark)
		style.SetFontSize(16)
		style.SetMargin(0, 0, 8, 0)
	})

	conversationDoc := ui.NewDocument(nil)
	conversationsView := ui.CreateDocumentView(conversationDoc)
	conversationsView.DisableScrollBlit = true
	conversationsView.Style = docStyle(false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(12)
		style.SetBackground(colorSidebarBG)
		style.SetOverflow(ui.OverflowAuto)
		style.SetScrollbarWidth(8)
		style.SetScrollbarTrack(0xECF1F7)
		style.SetScrollbarThumb(0xBAC7D6)
		style.SetScrollbarRadius(4)
		style.SetScrollbarPadding(1)
		style.SetContain(ui.ContainPaint)
	})
	conversationsView.StyleFocus = docStyle(false, func(style *ui.Style) {
		style.SetOutline(2, colorVKBlue)
		style.SetOutlineOffset(1)
	})
	app.conversationsView = conversationsView

	sidebar.Append(sidebarTitle)
	sidebar.Append(conversationsView)

	mainPanel := ui.CreateBox()
	applyStyle(mainPanel, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetPadding(10)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(14)
		style.SetBackground(colorPanelBG)
	})
	app.mainPanel = mainPanel

	conversationHead := elements.Label("Выберите диалог")
	applyStyle(conversationHead, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorVKBlueDark)
		style.SetFontSize(18)
		style.SetMargin(0, 0, 4, 0)
	})
	app.conversationHead = conversationHead

	conversationMeta := elements.Label("История сообщений появится после выбора диалога.")
	applyStyle(conversationMeta, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorMeta)
		style.SetFontSize(11)
		style.SetLineHeight(15)
		style.SetMargin(0, 0, 8, 0)
	})
	app.conversationMeta = conversationMeta

	messagesDoc := ui.NewDocument(nil)
	messagesView := ui.CreateDocumentView(messagesDoc)
	messagesView.DisableScrollBlit = true
	messagesView.Style = docStyle(false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(12)
		style.SetBackground(colorComposerBG)
		style.SetOverflow(ui.OverflowAuto)
		style.SetScrollbarWidth(8)
		style.SetScrollbarTrack(0xECF1F7)
		style.SetScrollbarThumb(0xBAC7D6)
		style.SetScrollbarRadius(4)
		style.SetScrollbarPadding(1)
		style.SetContain(ui.ContainPaint)
	})
	messagesView.StyleFocus = docStyle(false, func(style *ui.Style) {
		style.SetOutline(2, colorVKBlue)
		style.SetOutlineOffset(1)
	})
	app.messagesView = messagesView

	composeTitle := elements.Label("Новое сообщение")
	applyStyle(composeTitle, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorVKBlueDark)
		style.SetFontSize(14)
		style.SetMargin(10, 0, 4, 0)
	})

	composeInput := elements.Textarea("")
	applyStyle(composeInput, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(8)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(10)
		style.SetBackground(colorComposerBG)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetHeight(88)
		style.SetMargin(0, 0, 8, 0)
	})
	app.composeInput = composeInput

	sendButton := elements.Button("Отправить")
	styleActionButton(sendButton, colorVKBlueDark, ui.White, colorVKBlueDark)
	sendButton.OnClick = func() {
		_ = app.sendCurrentMessage()
	}
	app.sendButton = sendButton

	mainPanel.Append(conversationHead)
	mainPanel.Append(conversationMeta)
	mainPanel.Append(messagesView)
	mainPanel.Append(composeTitle)
	mainPanel.Append(composeInput)
	mainPanel.Append(sendButton)

	content.Append(sidebar)
	content.Append(mainPanel)

	shellRoot.Append(toolbar)
	shellRoot.Append(content)

	root.Append(header)
	root.Append(infoPanel)
	root.Append(authRoot)
	root.Append(shellRoot)
	window.Append(root)

	app.renderConversations()
	app.renderMessages()
	app.updateSendButton()
	app.handleResize(window.ClientRect())
}

func (app *App) handleResize(rect ui.Rect) {
	if !app.authorized {
		authWidth := rect.Width - windowPadding*2
		if authWidth > 460 {
			authWidth = 460
		}
		if authWidth < 300 {
			authWidth = 300
		}
		if app.authCard != nil {
			app.authCard.SetWidth(authWidth)
		}
		return
	}

	contentWidth := rect.Width - windowPadding*2
	if contentWidth < 700 {
		contentWidth = 700
	}
	sidebarWidth := sidebarDefaultWidth
	if rect.Width < 980 {
		sidebarWidth = sidebarCompactWidth
	}
	mainWidth := contentWidth - sidebarWidth - layoutGap
	if mainWidth < 360 {
		mainWidth = 360
		sidebarWidth = contentWidth - mainWidth - layoutGap
	}
	if sidebarWidth < 220 {
		sidebarWidth = 220
	}

	contentHeight := rect.Height - shellHeaderReservedHeight
	if contentHeight < contentMinimumHeight {
		contentHeight = contentMinimumHeight
	}
	conversationHeight := contentHeight - conversationHeaderHeight
	if conversationHeight < 160 {
		conversationHeight = 160
	}
	messageHeight := contentHeight - messageReservedHeight
	if messageHeight < 160 {
		messageHeight = 160
	}

	if app.sidebarPanel != nil {
		app.sidebarPanel.SetWidth(sidebarWidth)
		app.sidebarPanel.SetHeight(contentHeight)
	}
	if app.mainPanel != nil {
		app.mainPanel.SetWidth(mainWidth)
		app.mainPanel.SetHeight(contentHeight)
	}
	if app.conversationsView != nil {
		app.conversationsView.Style.SetHeight(conversationHeight)
		app.conversationsView.MarkLayoutDirty()
	}
	if app.messagesView != nil {
		app.messagesView.Style.SetHeight(messageHeight)
		app.messagesView.MarkLayoutDirty()
	}
	if app.composeInput != nil {
		app.composeInput.SetWidth(mainWidth - 24)
	}
	if app.sendButton != nil {
		app.sendButton.SetWidth(140)
	}
}

func (app *App) connectAndRefresh(paintBusy bool) error {
	settings, err := loadSession()
	if err != nil {
		app.currentUser = nil
		app.setProfile(nil)
		app.setStatus("Не удалось прочитать "+settingsConfigPath+": "+err.Error(), colorDanger)
		app.showAuthScreen()
		return err
	}
	app.session = settings
	if paintBusy {
		app.setStatus("Читаю settings.json и загружаю диалоги...", colorVKBlueDark)
		app.redrawNow()
	}
	client, err := app.clientFromSession()
	if err != nil {
		app.currentUser = nil
		app.setProfile(nil)
		app.setStatus(err.Error(), colorDanger)
		app.showAuthScreen()
		return err
	}
	self, err := client.Self()
	if err != nil {
		app.currentUser = nil
		app.setProfile(nil)
		app.setStatus(err.Error(), colorDanger)
		app.showAuthScreen()
		return err
	}
	app.currentUser = self
	app.setProfile(self)
	if err := app.refreshConversations(false, paintBusy); err != nil {
		return err
	}
	app.authorized = true
	app.showMessengerScreen()
	app.setStatus("Настройки загружены. Открываю первый диалог.", colorSuccess)
	return nil
}

func (app *App) refreshConversations(connectIfNeeded bool, paintBusy bool) error {
	client, err := app.clientFromSession()
	if err != nil {
		app.setStatus(err.Error(), colorDanger)
		return err
	}
	if connectIfNeeded && app.currentUser == nil {
		self, err := client.Self()
		if err != nil {
			app.setStatus(err.Error(), colorDanger)
			return err
		}
		app.currentUser = self
		app.setProfile(self)
	}
	if paintBusy {
		app.setStatus("Загружаю список диалогов...", colorVKBlueDark)
		app.redrawNow()
	}
	resp, err := client.Conversations(conversationPageSize)
	if err != nil {
		app.setStatus(err.Error(), colorDanger)
		return err
	}
	app.conversations = resp.Response.Items
	app.conversationUsers = usersByID(resp.Response.Profiles)
	app.conversationGroups = groupsByID(resp.Response.Groups)
	app.renderConversations()

	if len(app.conversations) == 0 {
		app.selectedPeerID = 0
		app.historyMessages = nil
		app.historyUsers = map[int64]vkUser{}
		app.historyGroups = map[int64]vkGroup{}
		app.syncConversationHeader()
		app.renderMessages()
		app.setStatus("Диалоги не найдены.", colorMeta)
		return nil
	}

	if app.selectedPeerID == 0 || app.findConversation(app.selectedPeerID) == nil {
		app.selectedPeerID = app.conversations[0].Conversation.Peer.ID
	}
	app.syncConversationHeader()
	app.renderConversations()
	if err := app.loadHistoryForSelection(paintBusy); err != nil {
		return err
	}
	app.setStatus("Диалоги обновлены: "+strconv.Itoa(len(app.conversations)), colorSuccess)
	return nil
}

func (app *App) loadHistoryForSelection(paintBusy bool) error {
	if app.selectedPeerID == 0 {
		app.setStatus("Сначала выберите диалог.", colorDanger)
		return fmt.Errorf("conversation not selected")
	}
	client, err := app.clientFromSession()
	if err != nil {
		app.setStatus(err.Error(), colorDanger)
		return err
	}
	if paintBusy {
		app.setStatus("Загружаю историю сообщений...", colorVKBlueDark)
		app.redrawNow()
	}
	resp, err := client.History(app.selectedPeerID, historyPageSize)
	if err != nil {
		app.setStatus(err.Error(), colorDanger)
		return err
	}
	app.historyMessages = resp.Response.Items
	app.historyUsers = usersByID(resp.Response.Profiles)
	app.historyGroups = groupsByID(resp.Response.Groups)
	app.renderMessages()
	app.syncConversationHeader()
	app.updateSendButton()
	app.setStatus("История загружена: "+strconv.Itoa(len(app.historyMessages))+" сообщений", colorSuccess)
	return nil
}

func (app *App) selectConversation(peerID int64) {
	if peerID == 0 || app.selectedPeerID == peerID {
		return
	}
	app.selectedPeerID = peerID
	app.syncConversationHeader()
	app.renderConversations()
	_ = app.loadHistoryForSelection(true)
}

func (app *App) sendCurrentMessage() error {
	current := app.findConversation(app.selectedPeerID)
	if current == nil {
		app.setStatus("Сначала выберите диалог.", colorDanger)
		return fmt.Errorf("conversation not selected")
	}
	if current.Conversation.CanWrite != nil && !current.Conversation.CanWrite.Allowed {
		app.setStatus("В этот диалог сейчас нельзя писать.", colorDanger)
		return fmt.Errorf("writing disabled")
	}
	message := strings.TrimSpace(app.composeInput.Text)
	if message == "" {
		app.setStatus("Сообщение пустое.", colorDanger)
		return fmt.Errorf("empty message")
	}
	client, err := app.clientFromSession()
	if err != nil {
		app.setStatus(err.Error(), colorDanger)
		return err
	}
	app.setStatus("Отправляю сообщение...", colorVKBlueDark)
	app.redrawNow()
	if _, err := client.Send(app.selectedPeerID, message); err != nil {
		app.setStatus(err.Error(), colorDanger)
		return err
	}
	app.composeInput.SetText(app.window, "")
	app.setStatus("Сообщение отправлено, обновляю диалог...", colorSuccess)
	if err := app.refreshConversations(false, false); err != nil {
		return err
	}
	return app.loadHistoryForSelection(false)
}

func (app *App) renderConversations() {
	root := docBox("conversation-root", func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(8)
		style.SetBackground(colorSidebarBG)
	})

	if len(app.conversations) == 0 {
		root.Append(emptyStateDocument(
			"Нет диалогов",
			"Подключите токен и загрузите список диалогов. Клиент читает и отправляет сообщения через VK API.",
		))
		app.conversationsView.SetDocument(ui.NewDocument(root))
		app.conversationsView.MarkLayoutDirty()
		return
	}

	for _, item := range app.conversations {
		peerID := item.Conversation.Peer.ID
		selected := peerID == app.selectedPeerID
		titleText := app.conversationTitle(item)
		if unread := item.Conversation.UnreadCount; unread > 0 {
			titleText = titleText + " (" + strconv.Itoa(unread) + ")"
		}
		card := docBox("conversation-card", func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 8, 0)
			style.SetPadding(8, 10)
			style.SetBorderRadius(10)
			style.SetBorder(1, colorPanelBorder)
			style.SetBackground(colorPanelBG)
			if selected {
				style.SetBackground(colorSelectedCard)
				style.SetBorderColor(colorSelectedBorder)
				style.SetBorderWidth(2)
			}
		},
			docText(titleText, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetForeground(colorText)
				style.SetFontSize(13)
				style.SetMargin(0, 0, 3, 0)
			}),
			docText(app.conversationPreview(item), func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetForeground(colorMeta)
				style.SetFontSize(11)
				style.SetLineHeight(15)
				style.SetMargin(0, 0, 4, 0)
			}),
			ui.NewDocumentText(app.conversationCardMeta(item), docStyle(true, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetForeground(colorVKBlueDark)
				style.SetFontSize(10)
			})),
		)
		card.Focusable = true
		card.StyleHover = docStyle(false, func(style *ui.Style) {
			style.SetBorderColor(colorSelectedBorder)
			style.SetBackground(colorSelectedCard)
		})
		card.StyleFocus = docStyle(false, func(style *ui.Style) {
			style.SetOutline(2, colorVKBlue)
			style.SetOutlineOffset(1)
			style.SetBorderColor(colorVKBlue)
		})
		attachDocumentClick(card, func(target int64) func() {
			return func() {
				app.selectConversation(target)
			}
		}(peerID))
		root.Append(card)
	}

	app.conversationsView.SetDocument(ui.NewDocument(root))
	app.conversationsView.MarkLayoutDirty()
}

func (app *App) renderMessages() {
	root := docBox("message-root", func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(10)
		style.SetBackground(colorComposerBG)
	})

	if app.selectedPeerID == 0 {
		root.Append(emptyStateDocument(
			"Диалог не выбран",
			"Слева выберите диалог. История загружается с последними сообщениями сверху, чтобы новый поток был виден сразу.",
		))
		app.messagesView.SetDocument(ui.NewDocument(root))
		app.messagesView.MarkLayoutDirty()
		return
	}

	root.Append(docText("Последние сообщения сверху", func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(colorMeta)
		style.SetFontSize(11)
		style.SetMargin(0, 0, 8, 0)
	}))

	if len(app.historyMessages) == 0 {
		root.Append(emptyStateDocument(
			"История пуста",
			"VK не вернул сообщений для выбранного диалога или они ещё не были загружены.",
		))
		app.messagesView.SetDocument(ui.NewDocument(root))
		app.messagesView.MarkLayoutDirty()
		return
	}

	for _, message := range app.historyMessages {
		outgoing := message.Out != 0
		sender := app.displayNameByOwnerID(message.FromID)
		meta := sender + " | " + formatTimestamp(message.Date)
		body := messageBody(&message)
		bubble := docBox("message-bubble", func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			if outgoing {
				style.SetMargin(0, 0, 10, 84)
				style.SetBackground(colorSoftBlue)
				style.SetBorderColor(colorSelectedBorder)
			} else {
				style.SetMargin(0, 84, 10, 0)
				style.SetBackground(colorSoftIncoming)
				style.SetBorderColor(colorPanelBorder)
			}
			style.SetPadding(8, 10)
			style.SetBorderWidth(1)
			style.SetBorderRadius(12)
		},
			ui.NewDocumentText(meta, docStyle(true, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetForeground(colorVKBlueDark)
				style.SetFontSize(10)
				style.SetMargin(0, 0, 4, 0)
			})),
			docText(body, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetForeground(colorText)
				style.SetFontSize(12)
				style.SetLineHeight(18)
				style.SetWhiteSpace(ui.WhiteSpacePreWrap)
			}),
		)
		root.Append(bubble)
	}

	app.messagesView.SetDocument(ui.NewDocument(root))
	app.messagesView.MarkLayoutDirty()
}

func (app *App) syncConversationHeader() {
	current := app.findConversation(app.selectedPeerID)
	if current == nil {
		app.conversationHead.SetText(app.window, "Выберите диалог")
		app.conversationMeta.SetText(app.window, "История сообщений появится после выбора диалога.")
		app.updateSendButton()
		return
	}
	app.conversationHead.SetText(app.window, app.conversationTitle(*current))
	app.conversationMeta.SetText(app.window, app.conversationMetaLine(*current))
	app.updateSendButton()
}

func (app *App) updateSendButton() {
	current := app.findConversation(app.selectedPeerID)
	if current == nil {
		app.sendButton.SetText(app.window, "Отправить")
		app.sendButton.UpdateStyle(func(style *ui.Style) {
			style.SetBackground(colorVKBlueDark)
			style.SetBorderColor(colorVKBlueDark)
		})
		return
	}
	if current.Conversation.CanWrite != nil && !current.Conversation.CanWrite.Allowed {
		app.sendButton.SetText(app.window, "Нельзя писать")
		app.sendButton.UpdateStyle(func(style *ui.Style) {
			style.SetBackground(0xB0BAC5)
			style.SetBorderColor(0x9AA5B1)
		})
		return
	}
	app.sendButton.SetText(app.window, "Отправить")
	app.sendButton.UpdateStyle(func(style *ui.Style) {
		style.SetBackground(colorVKBlueDark)
		style.SetBorderColor(colorVKBlueDark)
	})
}

// NETWORK

func (app *App) clientFromSession() (*vkClient, error) {
	token := strings.TrimSpace(app.session.AccessToken)
	if token == "" {
		return nil, fmt.Errorf("в %s не найден access_token", settingsConfigPath)
	}
	return &vkClient{
		baseURL:     normalizeAPIBase(app.session.APIBase),
		apiVersion:  normalizeAPIVersion(app.session.APIVersion),
		accessToken: token,
		httpClient:  app.httpClient,
	}, nil
}

func (app *App) showAuthScreen() {
	if app.window == nil {
		return
	}
	app.authorized = false
	if app.authRoot != nil {
		app.authRoot.UpdateStyle(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
		})
	}
	if app.shellRoot != nil {
		app.shellRoot.UpdateStyle(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayNone)
		})
	}
	app.applyWindowToDesktop()
	app.handleResize(app.window.ClientRect())
	app.redrawNow()
}

func (app *App) showMessengerScreen() {
	if app.window == nil {
		return
	}
	app.authorized = true
	if app.authRoot != nil {
		app.authRoot.UpdateStyle(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayNone)
		})
	}
	if app.shellRoot != nil {
		app.shellRoot.UpdateStyle(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
		})
	}
	app.applyWindowToDesktop()
	app.handleResize(app.window.ClientRect())
	app.redrawNow()
}

func (client *vkClient) Self() (*vkUser, error) {
	var response vkUserGetResponse
	params := url.Values{}
	params.Set("fields", "screen_name")
	if err := client.do("users.get", params, &response); err != nil {
		return nil, err
	}
	if len(response.Response) == 0 {
		return nil, fmt.Errorf("VK не вернул профиль пользователя")
	}
	user := response.Response[0]
	return &user, nil
}

func (client *vkClient) Conversations(count int) (*vkGetConversationsResponse, error) {
	var response vkGetConversationsResponse
	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("extended", "1")
	params.Set("fields", "screen_name")
	if err := client.do("messages.getConversations", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *vkClient) History(peerID int64, count int) (*vkGetHistoryResponse, error) {
	var response vkGetHistoryResponse
	params := url.Values{}
	params.Set("peer_id", strconv.FormatInt(peerID, 10))
	params.Set("count", strconv.Itoa(count))
	params.Set("extended", "1")
	params.Set("rev", "0")
	params.Set("fields", "screen_name")
	if err := client.do("messages.getHistory", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *vkClient) Send(peerID int64, message string) (int, error) {
	var envelope vkSendEnvelope
	params := url.Values{}
	params.Set("peer_id", strconv.FormatInt(peerID, 10))
	params.Set("message", message)
	params.Set("random_id", strconv.Itoa(int(time.Now().UnixNano()&0x7fffffff)))
	if err := client.do("messages.send", params, &envelope); err != nil {
		return 0, err
	}
	if len(envelope.Response) == 0 {
		return 0, fmt.Errorf("VK не вернул идентификатор сообщения")
	}

	var messageID int
	if err := json.Unmarshal(envelope.Response, &messageID); err == nil {
		return messageID, nil
	}
	var items []vkSendResponseItem
	if err := json.Unmarshal(envelope.Response, &items); err == nil && len(items) > 0 {
		if items[0].Error != nil {
			return 0, fmt.Errorf("VK send error %d: %s", items[0].Error.Code, strings.TrimSpace(items[0].Error.Description))
		}
		return items[0].MessageID, nil
	}
	return 0, fmt.Errorf("не удалось разобрать ответ messages.send")
}

func (client *vkClient) do(method string, params url.Values, target interface{}) error {
	data, err := client.doRaw(method, params)
	if err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	if err := safeJSONUnmarshal(data, target); err != nil {
		return fmt.Errorf("json decode %s: %w", method, err)
	}
	return nil
}

func (client *vkClient) doRaw(method string, params url.Values) ([]byte, error) {
	if client == nil || client.httpClient == nil {
		return nil, fmt.Errorf("HTTP client unavailable")
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", client.accessToken)
	params.Set("v", normalizeAPIVersion(client.apiVersion))

	request, err := http.NewRequest(http.MethodPost, normalizeAPIBase(client.baseURL)+method, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "KolibriOS VK Messenger/0.1")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %s", response.Status)
	}

	var apiError vkErrorEnvelope
	if err := safeJSONUnmarshal(data, &apiError); err == nil && apiError.Error != nil {
		return nil, apiError.Error
	}
	return data, nil
}

func safeJSONUnmarshal(data []byte, target interface{}) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while decoding JSON: %v | body=%s", recovered, shortResponseBody(data))
		}
	}()
	return json.Unmarshal(data, target)
}

func newHTTPClient(rootCAs *x509.CertPool) *http.Client {
	baseTransport, _ := http.DefaultTransport.(*http.Transport)
	transport := &http.Transport{}
	if baseTransport != nil {
		*transport = *baseTransport
		if baseTransport.TLSClientConfig != nil {
			transport.TLSClientConfig = baseTransport.TLSClientConfig.Clone()
		}
	}
	if rootCAs != nil {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		} else {
			transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		}
		transport.TLSClientConfig.RootCAs = rootCAs
	}
	return &http.Client{
		Transport: transport,
	}
}

func configureLocalCABundle() (string, bool) {
	if value := strings.TrimSpace(os.Getenv("SSL_CERT_FILE")); value != "" {
		return value, true
	}
	if _, err := os.Stat(localCABundlePath); err != nil {
		return "", false
	}
	if err := os.Setenv("SSL_CERT_FILE", localCABundlePath); err != nil {
		return "", false
	}
	return localCABundlePath, true
}

func loadRootPool(path string) (*x509.CertPool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("no certificates parsed from %s", path)
	}
	return pool, nil
}

func loadSession() (appSession, error) {
	session := appSession{
		APIBase:    defaultAPIBase,
		APIVersion: defaultAPIVersion,
	}
	if data, err := os.ReadFile(settingsConfigPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &session); err != nil {
			return session, err
		}
	}
	session.APIBase = normalizeAPIBase(session.APIBase)
	session.APIVersion = normalizeAPIVersion(session.APIVersion)
	return session, nil
}

func normalizeAPIBase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultAPIBase
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	value = strings.TrimRight(value, "/")
	if strings.HasSuffix(strings.ToLower(value), "/method") {
		return value + "/"
	}
	if !strings.HasSuffix(strings.ToLower(value), "/method/") {
		value += "/method"
	}
	return strings.TrimRight(value, "/") + "/"
}

func normalizeAPIVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultAPIVersion
	}
	return value
}

func desktopWindowBounds() (x int, y int, width int, height int) {
	work := kos.ScreenWorkingArea()
	width = work.Width()
	height = work.Height()
	x = work.Left
	y = work.Top
	if width <= 1 || height <= 1 {
		width, height = kos.ScreenSize()
		x = 0
		y = 0
	}
	if width < 1 {
		width = windowWidth
	}
	if height < 1 {
		height = windowHeight
	}
	return x, y, width, height
}

func (app *App) applyWindowToDesktop() {
	if app == nil || app.window == nil {
		return
	}
	x, y, width, height := desktopWindowBounds()
	app.window.SetBounds(x, y, width, height)
}

// DATA

func usersByID(items []vkUser) map[int64]vkUser {
	result := make(map[int64]vkUser, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func groupsByID(items []vkGroup) map[int64]vkGroup {
	result := make(map[int64]vkGroup, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func (app *App) findConversation(peerID int64) *vkConversationWithMessage {
	for index := 0; index < len(app.conversations); index++ {
		if app.conversations[index].Conversation.Peer.ID == peerID {
			return &app.conversations[index]
		}
	}
	return nil
}

func (app *App) conversationTitle(item vkConversationWithMessage) string {
	peer := item.Conversation.Peer
	switch peer.Type {
	case "chat":
		if item.Conversation.ChatSettings != nil && strings.TrimSpace(item.Conversation.ChatSettings.Title) != "" {
			return item.Conversation.ChatSettings.Title
		}
		if peer.LocalID != 0 {
			return "Чат #" + strconv.Itoa(peer.LocalID)
		}
		return "Чат " + strconv.FormatInt(peer.ID, 10)
	case "user":
		if user, ok := app.conversationUsers[peer.ID]; ok {
			return displayUserName(user)
		}
		return "Пользователь " + strconv.FormatInt(peer.ID, 10)
	case "group":
		groupID := peer.ID
		if groupID < 0 {
			groupID = -groupID
		}
		if group, ok := app.conversationGroups[groupID]; ok {
			return strings.TrimSpace(group.Name)
		}
		return "Сообщество " + strconv.FormatInt(groupID, 10)
	default:
		return "Peer " + strconv.FormatInt(peer.ID, 10)
	}
}

func (app *App) conversationPreview(item vkConversationWithMessage) string {
	prefix := ""
	if item.LastMessage.Out != 0 {
		prefix = "Вы: "
	} else if item.Conversation.Peer.Type == "chat" {
		prefix = app.displayNameByOwnerID(item.LastMessage.FromID) + ": "
	}
	return truncateText(prefix+messageBody(&item.LastMessage), 120)
}

func (app *App) conversationCardMeta(item vkConversationWithMessage) string {
	parts := []string{
		"type=" + item.Conversation.Peer.Type,
		"peer=" + strconv.FormatInt(item.Conversation.Peer.ID, 10),
		"at=" + formatTimestamp(item.LastMessage.Date),
	}
	if item.Conversation.UnreadCount > 0 {
		parts = append(parts, "unread="+strconv.Itoa(item.Conversation.UnreadCount))
	}
	if item.Conversation.ChatSettings != nil && item.Conversation.ChatSettings.MembersCount > 0 {
		parts = append(parts, "members="+strconv.Itoa(item.Conversation.ChatSettings.MembersCount))
	}
	return strings.Join(parts, " | ")
}

func (app *App) conversationMetaLine(item vkConversationWithMessage) string {
	parts := []string{
		"peer " + strconv.FormatInt(item.Conversation.Peer.ID, 10),
		"type " + item.Conversation.Peer.Type,
		"last " + formatTimestamp(item.LastMessage.Date),
	}
	if item.Conversation.UnreadCount > 0 {
		parts = append(parts, "непрочитано "+strconv.Itoa(item.Conversation.UnreadCount))
	}
	if item.Conversation.CanWrite != nil && !item.Conversation.CanWrite.Allowed {
		parts = append(parts, "write=no")
		if item.Conversation.CanWrite.Reason != 0 {
			parts = append(parts, "reason="+strconv.Itoa(item.Conversation.CanWrite.Reason))
		}
	} else {
		parts = append(parts, "write=yes")
	}
	return strings.Join(parts, " | ")
}

func (app *App) displayNameByOwnerID(ownerID int64) string {
	if ownerID == 0 {
		return "Система"
	}
	if ownerID > 0 {
		if user, ok := app.historyUsers[ownerID]; ok {
			return displayUserName(user)
		}
		if user, ok := app.conversationUsers[ownerID]; ok {
			return displayUserName(user)
		}
		if app.currentUser != nil && app.currentUser.ID == ownerID {
			return displayUserName(*app.currentUser)
		}
		return "id " + strconv.FormatInt(ownerID, 10)
	}
	groupID := -ownerID
	if group, ok := app.historyGroups[groupID]; ok {
		return strings.TrimSpace(group.Name)
	}
	if group, ok := app.conversationGroups[groupID]; ok {
		return strings.TrimSpace(group.Name)
	}
	return "club " + strconv.FormatInt(groupID, 10)
}

func displayUserName(user vkUser) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name != "" {
		return name
	}
	if user.ScreenName != "" {
		return "@" + strings.TrimSpace(user.ScreenName)
	}
	return "id " + strconv.FormatInt(user.ID, 10)
}

func messageBody(message *vkMessage) string {
	if message == nil {
		return ""
	}
	if message.Action != nil {
		return actionSummary(message.Action)
	}
	text := strings.TrimSpace(message.Text)
	attachmentText := attachmentSummary(message.Attachments)
	if text == "" {
		text = attachmentText
	} else if attachmentText != "" {
		text += "\n[" + attachmentText + "]"
	}
	if text == "" {
		text = "(пустое сообщение)"
	}
	if message.ReplyMessage != nil {
		text += "\n[reply]"
	}
	if forwarded := len(message.FwdMessages); forwarded > 0 {
		text += "\n[forwarded " + strconv.Itoa(forwarded) + "]"
	}
	return text
}

func actionSummary(action *vkMessageAction) string {
	if action == nil {
		return ""
	}
	switch action.Type {
	case "chat_create":
		if strings.TrimSpace(action.Text) != "" {
			return "Создан чат: " + strings.TrimSpace(action.Text)
		}
		return "Создан чат"
	case "chat_title_update":
		if strings.TrimSpace(action.Text) != "" {
			return "Изменено название: " + strings.TrimSpace(action.Text)
		}
		return "Изменено название чата"
	case "chat_invite_user":
		return "Приглашён участник " + strconv.FormatInt(action.MemberID, 10)
	case "chat_kick_user":
		return "Удалён участник " + strconv.FormatInt(action.MemberID, 10)
	case "chat_photo_update":
		return "Обновлена фотография чата"
	case "chat_photo_remove":
		return "Удалена фотография чата"
	case "chat_pin_message":
		return "Закреплено сообщение"
	case "chat_unpin_message":
		return "Откреплено сообщение"
	default:
		if strings.TrimSpace(action.Text) != "" {
			return action.Type + ": " + strings.TrimSpace(action.Text)
		}
		return action.Type
	}
}

// HELPERS

func attachmentSummary(items []vkMessageAttachment) string {
	if len(items) == 0 {
		return ""
	}
	names := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		name := strings.TrimSpace(item.Type)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}

func truncateText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes-3]) + "..."
}

func shortResponseBody(data []byte) string {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "<empty>"
	}
	return truncateText(text, 180)
}

func formatTimestamp(unixValue int64) string {
	if unixValue <= 0 {
		return "-"
	}
	return time.Unix(unixValue, 0).Format("02.01 15:04")
}

func (app *App) setStatus(text string, color kos.Color) {
	if app.statusLine == nil {
		return
	}
	app.statusLine.SetText(app.window, "Статус: "+strings.TrimSpace(text))
	app.statusLine.UpdateStyle(func(style *ui.Style) {
		style.SetForeground(color)
	})
}

func (app *App) setProfile(user *vkUser) {
	if app.profileLine == nil {
		return
	}
	if user == nil {
		app.profileLine.SetText(app.window, "Профиль: не авторизован")
		return
	}
	line := "Профиль: " + displayUserName(*user)
	if screenName := strings.TrimSpace(user.ScreenName); screenName != "" {
		line += " (@" + screenName + ")"
	}
	line += " | id " + strconv.FormatInt(user.ID, 10)
	app.profileLine.SetText(app.window, line)
}

func (app *App) redrawNow() {
	if app.started && app.window != nil {
		app.window.RedrawContent()
	}
}

func applyStyle(element *ui.Element, mono bool, update func(*ui.Style)) {
	element.UpdateStyle(func(style *ui.Style) {
		if mono {
			style.SetFontPath(monoFontPath)
		} else {
			style.SetFontPath(defaultFontPath)
		}
		if update != nil {
			update(style)
		}
	})
}

func styleActionButton(button *ui.Element, background kos.Color, foreground kos.Color, border kos.Color) {
	applyStyle(button, false, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 8, 0, 0)
		style.SetPadding(4, 12)
		style.SetBorderRadius(10)
		style.SetBorder(1, border)
		style.SetBackground(background)
		style.SetForeground(foreground)
		style.SetFontSize(12)
	})
}

func docStyle(mono bool, update func(*ui.Style)) ui.Style {
	style := ui.Style{}
	if mono {
		style.SetFontPath(monoFontPath)
	} else {
		style.SetFontPath(defaultFontPath)
	}
	if update != nil {
		update(&style)
	}
	return style
}

func docText(text string, update func(*ui.Style)) *ui.DocumentNode {
	return ui.NewDocumentText(text, docStyle(false, update))
}

func docBox(name string, update func(*ui.Style), children ...*ui.DocumentNode) *ui.DocumentNode {
	return ui.NewDocumentElement(name, docStyle(false, update), children...)
}

func emptyStateDocument(title string, detail string) *ui.DocumentNode {
	return docBox("empty-state", func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12, 14)
		style.SetBorder(1, colorPanelBorder)
		style.SetBorderRadius(12)
		style.SetBackground(colorPanelBG)
	},
		docText(title, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(colorVKBlueDark)
			style.SetFontSize(15)
			style.SetMargin(0, 0, 4, 0)
		}),
		docText(detail, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(colorMeta)
			style.SetFontSize(11)
			style.SetLineHeight(15)
		}),
	)
}

func attachDocumentClick(node *ui.DocumentNode, handler func()) {
	if node == nil || handler == nil {
		return
	}
	node.OnClick = handler
	for _, child := range node.Children {
		attachDocumentClick(child, handler)
	}
}

func main() {
	app := NewApp()
	app.Run()
}
