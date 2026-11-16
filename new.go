package cmsadmin

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/dracory/blockeditor"
	"github.com/dracory/cmsstore"
	"github.com/dracory/cmsadmin/shared"
	"github.com/samber/lo"
)

type AdminOptions struct {
	// BlockEditorDefinitions is the block definitions to use for the block editor
	// these definitions must also be registered in the frontend
	BlockEditorDefinitions []blockeditor.BlockDefinition

	// FuncLayout is an optional function to render the admin interface inside
	// your own layout (branding, menus, etc.). It receives the request and
	// response writer so the host project can access request context
	// (auth user, locale, etc.) when rendering the layout.
	// If nil, a default bare-bones HTML page is used (Bootstrap + Vue CDN).
	// Uses anonymous struct to match shopadmin/blogadmin/statsadmin exactly,
	// so consumers can reuse their layout function for cmsadmin.
	FuncLayout func(w http.ResponseWriter, r *http.Request, title string, body string, options struct {
		Styles     []string
		StyleURLs  []string
		Scripts    []string
		ScriptURLs []string
	}) string

	// Logger is the logger to use to log any errors. Optional
	Logger *slog.Logger

	// Store is the cmsstore.StoreInterface to use by the admin panel
	Store cmsstore.StoreInterface

	AdminHomeURL    string
	MediaManagerURL string
	PaddingTopPx    int
	PaddingRightPx  int
	PaddingBottomPx int
	PaddingLeftPx   int

	// flags holds a map of feature flags for internal use
	Flags map[string]bool
}

func New(options AdminOptions) (*admin, error) {
	if options.Store == nil {
		return nil, errors.New(shared.ERROR_STORE_IS_NIL)
	}

	if options.Logger == nil {
		return nil, errors.New(shared.ERROR_LOGGER_IS_NIL)
	}

	return &admin{
		blockEditorDefinitions: options.BlockEditorDefinitions,
		logger:                 options.Logger,
		store:                  options.Store,
		funcLayout:             options.FuncLayout,
		adminHomeURL:           options.AdminHomeURL,
		mediaManagerURL:        options.MediaManagerURL,
		paddingTopPx:           options.PaddingTopPx,
		paddingRightPx:         options.PaddingRightPx,
		paddingBottomPx:        options.PaddingBottomPx,
		paddingLeftPx:          options.PaddingLeftPx,
		flags:                  lo.Ternary(options.Flags != nil, options.Flags, map[string]bool{}),
	}, nil
}
