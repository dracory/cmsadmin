// Package cmsadmin provides a standalone CMS admin interface following
// the folder-per-controller pattern. Each controller is in its own
// subfolder and handles its own views and AJAX data.
//
// This module is modeled on github.com/dracory/shopadmin and
// github.com/dracory/blogadmin. It was extracted from
// github.com/dracory/cmsstore/admin; the dependency is one-way
// (cmsadmin -> cmsstore).
package cmsadmin

import (
	"context"
	"log/slog"
	"maps"
	"net/http"

	"github.com/dracory/blockeditor"
	adminBlocks "github.com/dracory/cmsadmin/blocks"
	adminMenus "github.com/dracory/cmsadmin/menus"
	adminPages "github.com/dracory/cmsadmin/pages"
	"github.com/dracory/cmsadmin/shared"
	adminSites "github.com/dracory/cmsadmin/sites"
	adminTemplates "github.com/dracory/cmsadmin/templates"
	adminTranslations "github.com/dracory/cmsadmin/translations"

	"github.com/dracory/cmsstore"
	"github.com/dracory/req"
)

// == TYPE ====================================================================

type admin struct {
	blockEditorDefinitions []blockeditor.BlockDefinition
	funcLayout             func(w http.ResponseWriter, r *http.Request, title string, body string, options struct {
		Styles     []string
		StyleURLs  []string
		Scripts    []string
		ScriptURLs []string
	}) string
	logger          *slog.Logger
	store           cmsstore.StoreInterface
	adminHomeURL    string
	mediaManagerURL string
	paddingTopPx    int
	paddingRightPx  int
	paddingBottomPx int
	paddingLeftPx   int
	flags           map[string]bool
}

// == INTERFACE IMPLEMENTATION CHECK ==========================================

var _ shared.AdminInterface = (*admin)(nil)

// == INTERFACE IMPLEMENTATION ================================================

func (a *admin) Handle(w http.ResponseWriter, r *http.Request) {
	path := req.GetStringTrimmedOr(r, "path", "home")

	if path == "" {
		path = shared.PathHome
	}

	ctx := context.WithValue(r.Context(), shared.KeyEndpoint, r.URL.Path)
	ctx = context.WithValue(ctx, shared.KeyAdminHomeURL, a.adminHomeURL)
	ctx = context.WithValue(ctx, shared.KeyMediaManagerURL, a.mediaManagerURL)
	ctx = context.WithValue(ctx, shared.KeyPaddingTopPx, a.paddingTopPx)
	ctx = context.WithValue(ctx, shared.KeyPaddingRightPx, a.paddingRightPx)
	ctx = context.WithValue(ctx, shared.KeyPaddingBottomPx, a.paddingBottomPx)
	ctx = context.WithValue(ctx, shared.KeyPaddingLeftPx, a.paddingLeftPx)

	routeFunc := a.getRoute(path)
	routeFunc(w, r.WithContext(ctx))
}

// == PRIVATE METHODS =========================================================

func (a *admin) getRoute(route string) func(w http.ResponseWriter, r *http.Request) {

	routes := map[string]func(w http.ResponseWriter, r *http.Request){
		shared.PathHome: a.pageHome,
	}

	maps.Copy(routes, a.blockRoutes())

	if a.store.MenusEnabled() {
		maps.Copy(routes, a.menuRoutes())
	}

	maps.Copy(routes, a.pageRoutes())
	maps.Copy(routes, a.siteRoutes())
	maps.Copy(routes, a.templateRoutes())

	if a.store.TranslationsEnabled() {
		maps.Copy(routes, a.translationRoutes())
	}

	if val, ok := routes[route]; ok {
		return val
	}

	return routes[shared.PathHome]
}

func (a *admin) render(w http.ResponseWriter, r *http.Request, webpageTitle, webpageHtml string, options struct {
	Styles     []string
	StyleURLs  []string
	Scripts    []string
	ScriptURLs []string
}) string {
	wrappedPageContent := shared.CMSContainer(r, webpageHtml)

	// If a custom layout is provided, try it first
	if a.funcLayout != nil {
		custom := a.funcLayout(w, r, webpageTitle, wrappedPageContent, options)
		if custom != "" {
			if w != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(custom))
				return ""
			}
			return custom
		}
	}

	webpage := shared.Layout(w, r, webpageTitle, wrappedPageContent, options)

	if w != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(webpage))
		return ""
	}

	return webpage
}

func (a *admin) blockRoutes() map[string]func(w http.ResponseWriter, r *http.Request) {
	blockRoutes := map[string]func(w http.ResponseWriter, r *http.Request){
		shared.PathBlocksBlockCreate:     adminBlocks.UI(a.uiConfig()).BlockCreate,
		shared.PathBlocksBlockDelete:     adminBlocks.UI(a.uiConfig()).BlockDelete,
		shared.PathBlocksBlockManager:    adminBlocks.UI(a.uiConfig()).BlockManager,
		shared.PathBlocksBlockUpdate:     adminBlocks.UI(a.uiConfig()).BlockUpdate,
		shared.PathBlocksBlockVersioning: adminBlocks.UI(a.uiConfig()).BlockVersioning,
	}
	return blockRoutes
}

func (a *admin) menuRoutes() map[string]func(w http.ResponseWriter, r *http.Request) {
	menuRoutes := map[string]func(w http.ResponseWriter, r *http.Request){
		shared.PathMenusMenuCreate:     adminMenus.UI(a.uiConfig()).MenuCreate,
		shared.PathMenusMenuDelete:     adminMenus.UI(a.uiConfig()).MenuDelete,
		shared.PathMenusMenuManager:    adminMenus.UI(a.uiConfig()).MenuManager,
		shared.PathMenusMenuUpdate:     adminMenus.UI(a.uiConfig()).MenuUpdate,
		shared.PathMenusMenuVersioning: adminMenus.UI(a.uiConfig()).MenuVersioning,
	}
	return menuRoutes
}

func (a *admin) pageRoutes() map[string]func(w http.ResponseWriter, r *http.Request) {
	pageRoutes := map[string]func(w http.ResponseWriter, r *http.Request){
		shared.PathPagesPageCreate:     adminPages.UI(a.uiConfig()).PageCreate,
		shared.PathPagesPageDelete:     adminPages.UI(a.uiConfig()).PageDelete,
		shared.PathPagesPageManager:    adminPages.UI(a.uiConfig()).PageManager,
		shared.PathPagesPageUpdate:     adminPages.UI(a.uiConfig()).PageUpdate,
		shared.PathPagesPageVersioning: adminPages.UI(a.uiConfig()).PageVersioning,
	}
	return pageRoutes
}

func (a *admin) siteRoutes() map[string]func(w http.ResponseWriter, r *http.Request) {
	siteRoutes := map[string]func(w http.ResponseWriter, r *http.Request){
		shared.PathSitesSiteCreate:     adminSites.UI(a.uiConfig()).SiteCreate,
		shared.PathSitesSiteDelete:     adminSites.UI(a.uiConfig()).SiteDelete,
		shared.PathSitesSiteUpdate:     adminSites.UI(a.uiConfig()).SiteUpdate,
		shared.PathSitesSiteVersioning: adminSites.UI(a.uiConfig()).SiteVersioning,
		shared.PathSitesSiteManager:    adminSites.UI(a.uiConfig()).SiteManager,
	}

	return siteRoutes
}

func (a *admin) templateRoutes() map[string]func(w http.ResponseWriter, r *http.Request) {
	templateRoutes := map[string]func(w http.ResponseWriter, r *http.Request){
		shared.PathTemplatesTemplateCreate:     adminTemplates.UI(a.uiConfig()).TemplateCreate,
		shared.PathTemplatesTemplateDelete:     adminTemplates.UI(a.uiConfig()).TemplateDelete,
		shared.PathTemplatesTemplateManager:    adminTemplates.UI(a.uiConfig()).TemplateManager,
		shared.PathTemplatesTemplateUpdate:     adminTemplates.UI(a.uiConfig()).TemplateUpdate,
		shared.PathTemplatesTemplateVersioning: adminTemplates.UI(a.uiConfig()).TemplateVersioning,
	}
	return templateRoutes
}

func (a *admin) translationRoutes() map[string]func(w http.ResponseWriter, r *http.Request) {
	translationsRoutes := map[string]func(w http.ResponseWriter, r *http.Request){
		shared.PathTranslationsTranslationCreate:     adminTranslations.UI(a.uiConfig()).TranslationCreate,
		shared.PathTranslationsTranslationDelete:     adminTranslations.UI(a.uiConfig()).TranslationDelete,
		shared.PathTranslationsTranslationManager:    adminTranslations.UI(a.uiConfig()).TranslationManager,
		shared.PathTranslationsTranslationUpdate:     adminTranslations.UI(a.uiConfig()).TranslationUpdate,
		shared.PathTranslationsTranslationVersioning: adminTranslations.UI(a.uiConfig()).TranslationVersioning,
	}
	return translationsRoutes
}

// func (a *admin) adminBreadcrumbs(r *http.Request, pageBreadcrumbs []shared.Breadcrumb) hb.TagInterface {
// 	return shared.AdminBreadcrumbs(r, pageBreadcrumbs)
// }

func (a *admin) uiConfig() shared.UiConfig {
	return shared.UiConfig{
		BlockEditorDefinitions: a.blockEditorDefinitions,
		Layout:                 a.render,
		Logger:                 a.logger,
		Store:                  a.store,
	}
}
