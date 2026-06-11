package main

/*
#cgo pkg-config: gtk+-2.0 x11

#include <gtk/gtk.h>
#include <gdk/gdk.h>
#include <X11/Xlib.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

extern void onApplyBrightness();
extern void onQuitClicked();
extern void onSwipeStart(double x, double y);
extern void onSwipeEnd(double x, double y);
extern void onToggleEntityClicked(char *entity);
extern void onMacroActionClicked(char *action);
extern void processUIQueue();

// Style engine
// Apply a tailored RC style for buttons on e-ink displays
static void w_apply_button_style() {
    gtk_rc_parse_string(
        // ── Base button style ──
        // White/near-white bg by default, dark bg when pressed — e-ink friendly.
        "style \"kindle-btn\"\n"
        "{\n"
        "  xthickness = 2\n"
        "  ythickness = 2\n"
        "  font_name = \"sans bold 10\"\n"
        "  bg[NORMAL] = {0.95, 0.95, 0.95}\n"
        "  bg[PRELIGHT] = {0.95, 0.95, 0.95}\n"
        "  bg[ACTIVE] = {0.25, 0.25, 0.25}\n"
        "  fg[NORMAL] = {0.10, 0.10, 0.10}\n"
        "  fg[PRELIGHT] = {0.10, 0.10, 0.10}\n"
        "  fg[ACTIVE] = {1.0, 1.0, 1.0}\n"
        "}\n"
        "widget_class \"*<GtkButton>\" style \"kindle-btn\"\n"
        "\n"
        // ── Primary action buttons (Apply, media transport) ──
        "style \"kindle-btn-primary\" = \"kindle-btn\"\n"
        "{\n"
        "  xthickness = 2\n"
        "  ythickness = 2\n"
        "  font_name = \"sans 10 bold\"\n"
        "  bg[NORMAL] = {0.92, 0.92, 0.92}\n"
        "  bg[PRELIGHT] = {0.92, 0.92, 0.92}\n"
        "  bg[ACTIVE] = {0.20, 0.20, 0.20}\n"
        "  fg[NORMAL] = {0.08, 0.08, 0.08}\n"
        "  fg[PRELIGHT] = {0.08, 0.08, 0.08}\n"
        "  fg[ACTIVE] = {1.0, 1.0, 1.0}\n"
        "}\n"
        "widget \"*apply-btn*\" style \"kindle-btn-primary\"\n"
        "widget \"*media-btn*\" style \"kindle-btn-primary\"\n"
        // ── Toggle (light entity) buttons ──
        "style \"kindle-btn-toggle\" = \"kindle-btn\"\n"
        "{\n"
        "  font_name = \"sans 11 bold\"\n"
        "  xthickness = 8\n"
        "  ythickness = 8\n"
        "  bg[NORMAL] = {0.94, 0.94, 0.94}\n"
        "  bg[PRELIGHT] = {0.94, 0.94, 0.94}\n"
        "  bg[ACTIVE] = {0.22, 0.22, 0.22}\n"
        "  fg[NORMAL] = {0.10, 0.10, 0.10}\n"
        "  fg[PRELIGHT] = {0.10, 0.10, 0.10}\n"
        "  fg[ACTIVE] = {1.0, 1.0, 1.0}\n"
        "}\n"
        "widget \"*toggle-btn*\" style \"kindle-btn-toggle\"\n"
        "\n"
        // ── Frame / card style ──
        "style \"kindle-frame\"\n"
        "{\n"
        "  xthickness = 2\n"
        "  ythickness = 2\n"
        "  bg[NORMAL] = {0.92, 0.92, 0.92}\n"
        "  fg[NORMAL] = {0.15, 0.15, 0.15}\n"
        "}\n"
        "widget_class \"*<GtkFrame>\" style \"kindle-frame\"\n"
    );
}

// ── Icon renderer ──
// Draws a simple 24×24 icon for a macro button using Cairo onto a surface,
// then converts to GdkPixbuf via raw data copy.
// Returns a GtkImage widget ready to pass to gtk_button_set_image().
static GtkWidget* w_make_icon(const char *name) {
    const int S = 24;
    cairo_surface_t *surf = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, S, S);
    cairo_t *cr = cairo_create(surf);

    // Clear to transparent
    cairo_set_source_rgba(cr, 0, 0, 0, 0);
    cairo_paint(cr);

    // Ink colour — dark for e-ink contrast
    cairo_set_source_rgb(cr, 0.08, 0.08, 0.08);
    cairo_set_line_width(cr, 2.0);
    cairo_set_line_cap(cr, CAIRO_LINE_CAP_ROUND);
    cairo_set_line_join(cr, CAIRO_LINE_JOIN_ROUND);

    // ── Draw icon shapes ──
    if (strcmp(name, "prev_track") == 0) {
        cairo_move_to(cr, 18, 5); cairo_line_to(cr, 8, 12); cairo_line_to(cr, 18, 19); cairo_close_path(cr); cairo_fill(cr);
        cairo_rectangle(cr, 5, 4, 2, 16); cairo_fill(cr);
    } else if (strcmp(name, "play_pause") == 0) {
        cairo_move_to(cr, 7, 4); cairo_line_to(cr, 19, 12); cairo_line_to(cr, 7, 20); cairo_close_path(cr); cairo_fill(cr);
    } else if (strcmp(name, "pause") == 0 || strcmp(name, "paused") == 0) {
        cairo_rectangle(cr, 6, 4, 4, 16); cairo_fill(cr);
        cairo_rectangle(cr, 14, 4, 4, 16); cairo_fill(cr);
    } else if (strcmp(name, "next_track") == 0) {
        cairo_move_to(cr, 6, 5); cairo_line_to(cr, 16, 12); cairo_line_to(cr, 6, 19); cairo_close_path(cr); cairo_fill(cr);
        cairo_rectangle(cr, 17, 4, 2, 16); cairo_fill(cr);
    } else if (strcmp(name, "mute_mic") == 0) {
        cairo_rectangle(cr, 8, 4, 6, 8); cairo_stroke(cr);
        cairo_move_to(cr, 11, 12); cairo_line_to(cr, 11, 17); cairo_stroke(cr);
        cairo_arc(cr, 11, 17, 4, 3.1416, 0); cairo_stroke(cr);
        cairo_move_to(cr, 4, 4); cairo_line_to(cr, 18, 20); cairo_stroke(cr);
    } else if (strcmp(name, "monitor_toggle") == 0 || strcmp(name, "monitor_off") == 0) {
        cairo_rectangle(cr, 3, 3, 18, 12); cairo_stroke(cr);
        cairo_move_to(cr, 8, 17); cairo_line_to(cr, 16, 17); cairo_stroke(cr);
        cairo_move_to(cr, 12, 15); cairo_line_to(cr, 12, 17); cairo_stroke(cr);
    } else if (strcmp(name, "monitor_on") == 0) {
        cairo_rectangle(cr, 3, 3, 18, 12); cairo_fill(cr);
        cairo_move_to(cr, 8, 17); cairo_line_to(cr, 16, 17); cairo_stroke(cr);
        cairo_move_to(cr, 12, 15); cairo_line_to(cr, 12, 17); cairo_stroke(cr);
    } else if (strcmp(name, "pc_mode_toggle") == 0 || strcmp(name, "pc_mode_power") == 0) {
        cairo_arc(cr, 12, 12, 7, 0, 6.2832); cairo_move_to(cr, 10, 12); cairo_line_to(cr, 10, 14); cairo_move_to(cr, 9, 13); cairo_line_to(cr, 11, 13); cairo_stroke(cr);
        cairo_arc(cr, 16, 10, 1.5, 0, 6.2832); cairo_fill(cr);
        cairo_arc(cr, 17, 14, 1.5, 0, 6.2832); cairo_fill(cr);
    } else if (strcmp(name, "pc_mode_save") == 0) {
        cairo_arc(cr, 12, 12, 7, 0, 6.2832); cairo_stroke(cr);
        cairo_move_to(cr, 12, 7); cairo_line_to(cr, 12, 12); cairo_line_to(cr, 16, 14); cairo_stroke(cr);
    } else if (strcmp(name, "launch_chrome") == 0) {
        cairo_arc(cr, 12, 12, 9, 0, 6.2832); cairo_stroke(cr);
        cairo_arc(cr, 12, 12, 4, 0, 6.2832); cairo_stroke(cr);
        cairo_move_to(cr, 3, 12); cairo_line_to(cr, 21, 12); cairo_stroke(cr);
        cairo_move_to(cr, 12, 3); cairo_line_to(cr, 12, 21); cairo_stroke(cr);
    } else if (strcmp(name, "launch_mail") == 0) {
        cairo_rectangle(cr, 2, 5, 20, 14); cairo_stroke(cr);
        cairo_move_to(cr, 2, 6); cairo_line_to(cr, 12, 14); cairo_line_to(cr, 22, 6); cairo_stroke(cr);
    } else if (strcmp(name, "sleep") == 0) {
        cairo_arc(cr, 14, 12, 8, 0, 6.2832); cairo_stroke(cr);
        cairo_arc(cr, 10, 9, 6, -1.2, 1.2); cairo_fill(cr);
    } else if (strcmp(name, "restart") == 0) {
        cairo_arc(cr, 12, 12, 8, 0.5, 5.8); cairo_stroke(cr);
        cairo_move_to(cr, 15, 6); cairo_line_to(cr, 20, 8); cairo_line_to(cr, 18, 13); cairo_stroke(cr);
    } else if (strcmp(name, "shutdown") == 0) {
        cairo_arc(cr, 12, 12, 8, 0.8, 5.5); cairo_stroke(cr);
        cairo_move_to(cr, 12, 3); cairo_line_to(cr, 12, 13); cairo_stroke(cr);
    } else if (strcmp(name, "launch_fortnite") == 0) {
        cairo_arc(cr, 12, 12, 9, 0, 6.2832); cairo_stroke(cr);
        cairo_arc(cr, 12, 12, 3, 0, 6.2832); cairo_stroke(cr);
        cairo_move_to(cr, 12, 1); cairo_line_to(cr, 12, 5); cairo_stroke(cr);
        cairo_move_to(cr, 12, 19); cairo_line_to(cr, 12, 23); cairo_stroke(cr);
        cairo_move_to(cr, 1, 12); cairo_line_to(cr, 5, 12); cairo_stroke(cr);
        cairo_move_to(cr, 19, 12); cairo_line_to(cr, 23, 12); cairo_stroke(cr);
    } else {
        cairo_arc(cr, 12, 12, 8, 0, 6.2832); cairo_stroke(cr);
        cairo_arc(cr, 12, 12, 2, 0, 6.2832); cairo_fill(cr);
    }

    cairo_destroy(cr);

    // Convert surface to GdkPixbuf
    int stride = cairo_image_surface_get_stride(surf);
    unsigned char *pixels = cairo_image_surface_get_data(surf);
    GdkPixbuf *pb = gdk_pixbuf_new_from_data(pixels, GDK_COLORSPACE_RGB, TRUE, 8, S, S, stride, NULL, NULL);
    if (!pb) { cairo_surface_destroy(surf); return gtk_image_new_from_stock(GTK_STOCK_MEDIA_PLAY, GTK_ICON_SIZE_BUTTON); }
    // Copy so we can destroy the surface
    GdkPixbuf *copy = gdk_pixbuf_copy(pb);
    g_object_unref(pb);
    cairo_surface_destroy(surf);

    GtkWidget *img = gtk_image_new_from_pixbuf(copy);
    g_object_unref(copy);
    return img;
}

// Widget factories
// Call this before any GTK/GLib usage for GTK 2.10 compat
static void w_init() { g_type_init(); w_apply_button_style(); }

static GtkWidget* w_win(gboolean hw_landscape) {
	GtkWidget *w = gtk_window_new(GTK_WINDOW_TOPLEVEL);
	if (hw_landscape) {
		// Kindle winmgr parses the title as window params.  O:LR declares that
		// this application supports landscape orientations, which lets winmgr
		// call Lab126's native ligl screen rotation path.
		gtk_window_set_title(GTK_WINDOW(w), "L:A_N:application_ID:kindle-dashboard_module:dashboard_O:LR_PC:N");
	} else {
		gtk_window_set_title(GTK_WINDOW(w), "Kindle Dashboard");
		gtk_window_set_keep_above(GTK_WINDOW(w), TRUE);
	}
	gtk_window_set_resizable(GTK_WINDOW(w), FALSE);
	gtk_window_set_decorated(GTK_WINDOW(w), FALSE);
	gtk_window_move(GTK_WINDOW(w), 0, 0);
	// Force exact size
	GdkGeometry geom;
	if (hw_landscape) {
		geom.min_width = 800; geom.max_width = 800;
		geom.min_height = 600; geom.max_height = 600;
	} else {
		geom.min_width = 600; geom.max_width = 600;
		geom.min_height = 800; geom.max_height = 800;
	}
	gtk_window_set_geometry_hints(GTK_WINDOW(w), NULL, &geom,
		GDK_HINT_MIN_SIZE | GDK_HINT_MAX_SIZE);
	return w;
}
static GtkWidget* w_vbox(gboolean h, gint s) { return gtk_vbox_new(h, s); }
static GtkWidget* w_hbox(gboolean h, gint s) { return gtk_hbox_new(h, s); }
static GtkWidget* w_frame(const char *l)     { return gtk_frame_new(l); }
static GtkWidget* w_lbl()                    { GtkWidget *l = gtk_label_new(NULL); return l; }
static GtkWidget* w_btn(const char *t)       { return gtk_button_new_with_label(t); }
static GtkWidget* w_btn_named(const char *t, const char *n) {
    GtkWidget *b = gtk_button_new_with_label(t);
    gtk_widget_set_name(b, n);
    return b;
}
static void w_btn_set_icon(GtkWidget *btn, const char *icon) {
    GtkWidget *img = w_make_icon(icon);
    gtk_button_set_image(GTK_BUTTON(btn), img);
}
static GtkWidget* w_btn_icon(const char *icon, const char *n) {
    GtkWidget *b = gtk_button_new();
    gtk_widget_set_name(b, n);
    w_btn_set_icon(b, icon);
    return b;
}
static void w_btn_text(GtkWidget *b, const char *t) { gtk_button_set_label(GTK_BUTTON(b), t); }
static GtkWidget* w_hscale(double min, double max, double step) {
	GtkAdjustment *a = (GtkAdjustment*)gtk_adjustment_new(min, min, max, step, step*10, 0);
	GtkWidget *s = gtk_hscale_new(GTK_ADJUSTMENT(a));
	gtk_scale_set_draw_value(GTK_SCALE(s), FALSE);
	return s;
}
static GtkWidget* w_table(gint r, gint c) {
	GtkWidget *t = gtk_table_new(r, c, TRUE);
	gtk_table_set_row_spacings(GTK_TABLE(t), 0);
	gtk_table_set_col_spacings(GTK_TABLE(t), 0);
	return t;
}

// Helpers
static void w_markup(GtkWidget *l, const char *m) { gtk_label_set_markup(GTK_LABEL(l), m); }
static void w_text(GtkWidget *l, const char *t)   { gtk_label_set_text(GTK_LABEL(l), t); }
static void w_align(GtkWidget *l, float x, float y) { gtk_misc_set_alignment(GTK_MISC(l), x, y); }
static void w_pack(GtkWidget *b, GtkWidget *c, gboolean ex, gboolean fi, guint pa) {
	gtk_box_pack_start(GTK_BOX(b), c, ex, fi, pa);
}
static void w_pack_end(GtkWidget *b, GtkWidget *c, gboolean ex, gboolean fi, guint pa) {
	gtk_box_pack_end(GTK_BOX(b), c, ex, fi, pa);
}
static void w_add(GtkWidget *p, GtkWidget *c) { gtk_container_add(GTK_CONTAINER(p), c); }
static void w_border(GtkWidget *w, guint px)  { gtk_container_set_border_width(GTK_CONTAINER(w), px); }
static void w_size(GtkWidget *w, gint x, gint y) { gtk_widget_set_size_request(w, x, y); }
static void w_redraw(GtkWidget *w) { gtk_widget_queue_draw(w); }
static void w_shadow(GtkWidget *w, GtkShadowType s) { gtk_frame_set_shadow_type(GTK_FRAME(w), s); }
static void w_table_put(GtkWidget *t, GtkWidget *w, gint l, gint r, gint tp, gint bt) {
	gtk_table_attach(GTK_TABLE(t), w, l, r, tp, bt, GTK_EXPAND|GTK_FILL, GTK_EXPAND|GTK_FILL, 0, 0);
}
static void w_table_put_center(GtkWidget *t, GtkWidget *w, gint l, gint r, gint tp, gint bt) {
	gtk_table_attach(GTK_TABLE(t), w, l, r, tp, bt, GTK_EXPAND, GTK_EXPAND, 0, 0);
}
static void w_table_spacing(GtkWidget *t, guint rs, guint cs) {
	gtk_table_set_row_spacings(GTK_TABLE(t), rs);
	gtk_table_set_col_spacings(GTK_TABLE(t), cs);
}
static void w_fg(GtkWidget *w, const char *c) {
	GdkColor col; gdk_color_parse(c, &col);
	gtk_widget_modify_fg(w, GTK_STATE_NORMAL, &col);
	gtk_widget_modify_fg(w, GTK_STATE_ACTIVE, &col);
	gtk_widget_modify_fg(w, GTK_STATE_PRELIGHT, &col);
}
static void w_bg_all(GtkWidget *w, const char *c) {
	GdkColor col; gdk_color_parse(c, &col);
	gtk_widget_modify_bg(w, GTK_STATE_NORMAL, &col);
	gtk_widget_modify_bg(w, GTK_STATE_ACTIVE, &col);
	gtk_widget_modify_bg(w, GTK_STATE_PRELIGHT, &col);
}
// Set label foreground for ALL states (NORMAL, ACTIVE, PRELIGHT) on the label
// child of a button. The label widget has its own GtkStyle separate from the
// button — setting just the button's fg doesn't propagate to it.
static void w_set_btn_label_fg(GtkWidget *btn, const char *c) {
	GtkWidget *child = gtk_bin_get_child(GTK_BIN(btn));
	if (!child) return;
	GdkColor col; gdk_color_parse(c, &col);
	gtk_widget_modify_fg(child, GTK_STATE_NORMAL, &col);
	gtk_widget_modify_fg(child, GTK_STATE_ACTIVE, &col);
	gtk_widget_modify_fg(child, GTK_STATE_PRELIGHT, &col);
}
// Set ONLY the ACTIVE foreground on a button's label child to white.
// PRELIGHT (hover) intentionally omitted — no hover effect on e-ink.
// This ensures the label text is always readable against the dark ACTIVE background.
static void w_set_btn_label_fg_active_white(GtkWidget *btn) {
	GtkWidget *child = gtk_bin_get_child(GTK_BIN(btn));
	if (!child) return;
	GdkColor col;
	gdk_color_parse("#ffffff", &col);
	gtk_widget_modify_fg(child, GTK_STATE_ACTIVE, &col);
}
static void w_bg(GtkWidget *w, const char *c) {
	GdkColor col; gdk_color_parse(c, &col);
	gtk_widget_modify_bg(w, GTK_STATE_NORMAL, &col);
}
static void w_btn_bg(GtkWidget *b, const char *c) {
	GdkColor col; gdk_color_parse(c, &col);
	gtk_widget_modify_bg(b, GTK_STATE_NORMAL, &col);
	gtk_widget_modify_bg(b, GTK_STATE_PRELIGHT, &col);
}
static void w_btn_markup(GtkWidget *b, const char *m) {
	GtkWidget *child = gtk_bin_get_child(GTK_BIN(b));
	if (child && GTK_IS_LABEL(child)) gtk_label_set_markup(GTK_LABEL(child), m);
}
static void w_wrap(GtkWidget *l) { gtk_label_set_line_wrap(GTK_LABEL(l), TRUE); }
static double w_scale_get(GtkWidget *s) { return gtk_range_get_value(GTK_RANGE(s)); }
static void w_scale_set(GtkWidget *s, double v) { gtk_range_set_value(GTK_RANGE(s), v); }
static void w_signal(GtkWidget *w, const char *s, GCallback cb) {
	g_signal_connect(G_OBJECT(w), s, cb, NULL);
}
static void w_show(GtkWidget *w)   { gtk_widget_show(w); }
static void w_hide(GtkWidget *w)   { gtk_widget_hide(w); }
static void w_show_all(GtkWidget *w) { gtk_widget_show_all(w); }

static gboolean w_is_swipe_target(GdkEventButton *event) {
	GtkWidget *target = gtk_get_event_widget((GdkEvent*)event);
	if (!target) return TRUE;
	if (GTK_IS_BUTTON(target) || GTK_IS_RANGE(target)) return FALSE;
	if (gtk_widget_get_ancestor(target, GTK_TYPE_BUTTON) != NULL) return FALSE;
	if (gtk_widget_get_ancestor(target, GTK_TYPE_RANGE) != NULL) return FALSE;
	return TRUE;
}

static gboolean w_on_button_press(GtkWidget *widget, GdkEventButton *event, gpointer data) {
	if (!event || event->button != 1 || !w_is_swipe_target(event)) return FALSE;
	onSwipeStart(event->x_root, event->y_root);
	return FALSE;
}

static gboolean w_on_button_release(GtkWidget *widget, GdkEventButton *event, gpointer data) {
	if (!event || event->button != 1) return FALSE;
	onSwipeEnd(event->x_root, event->y_root);
	return FALSE;
}

static void w_enable_swipe(GtkWidget *w) {
	gtk_widget_add_events(w, GDK_BUTTON_PRESS_MASK | GDK_BUTTON_RELEASE_MASK);
	g_signal_connect(G_OBJECT(w), "button-press-event", G_CALLBACK(w_on_button_press), NULL);
	g_signal_connect(G_OBJECT(w), "button-release-event", G_CALLBACK(w_on_button_release), NULL);
}

static void w_on_toggle_entity(GtkWidget *widget, gpointer data) {
	if (!data) return;
	onToggleEntityClicked((char*)data);
}

static void w_bind_toggle(GtkWidget *w, const char *entity) {
	char *copy = g_strdup(entity);
	g_signal_connect_data(G_OBJECT(w), "clicked", G_CALLBACK(w_on_toggle_entity), copy, (GClosureNotify)g_free, 0);
}

static void w_on_macro_action(GtkWidget *widget, gpointer data) {
	if (!data) return;
	onMacroActionClicked((char*)data);
}

static void w_bind_macro(GtkWidget *w, const char *action) {
	char *copy = g_strdup(action);
	g_signal_connect_data(G_OBJECT(w), "clicked", G_CALLBACK(w_on_macro_action), copy, (GClosureNotify)g_free, 0);
}

static gboolean w_ui_idle(gpointer data) {
	processUIQueue();
	return FALSE;
}

static void w_queue_ui_dispatch(void) {
	g_idle_add(w_ui_idle, NULL);
}

// Override-redirect to bypass Kindle window manager
static void w_override(void) {
	Display *dpy = XOpenDisplay(NULL);
	if (!dpy) return;
	Window root = DefaultRootWindow(dpy);
	Window par, *kids;
	unsigned int n;
	if (XQueryTree(dpy, root, &root, &par, &kids, &n)) {
		for (unsigned int i = 0; i < n; i++) {
			char *nm = NULL;
			if (XFetchName(dpy, kids[i], &nm) && nm) {
				if (strstr(nm, "Kindle Dashboard")) {
					XSetWindowAttributes a;
					a.override_redirect = True;
					XChangeWindowAttributes(dpy, kids[i], CWOverrideRedirect, &a);
					XMapRaised(dpy, kids[i]);
					XFree(nm); break;
				}
				XFree(nm);
			}
		}
		if (kids) XFree(kids);
	}
	XFlush(dpy); XCloseDisplay(dpy);
}
*/
import "C"
import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Button name constants for RC style targeting
const (
	btnNameMedia  = "media-btn"
	btnNameApply  = "apply-btn"
	btnNameToggle = "toggle-btn"
)

type ViewID int

const (
	ViewCalendar ViewID = iota
	ViewHome
	ViewLauncher
	ViewInfo
	viewCount
)

// ─── State ───
var dash *Dashboard
var hassClient *HassClient
var pcMacroClient *PCMacroClient
var uiQueue struct {
	mu        sync.Mutex
	items     []func()
	scheduled bool
}

func enqueueUI(fn func()) {
	uiQueue.mu.Lock()
	uiQueue.items = append(uiQueue.items, fn)
	if uiQueue.scheduled {
		uiQueue.mu.Unlock()
		return
	}
	uiQueue.scheduled = true
	uiQueue.mu.Unlock()
	C.w_queue_ui_dispatch()
}

//export processUIQueue
func processUIQueue() {
	for {
		uiQueue.mu.Lock()
		if len(uiQueue.items) == 0 {
			uiQueue.scheduled = false
			uiQueue.mu.Unlock()
			return
		}
		items := uiQueue.items
		uiQueue.items = nil
		uiQueue.mu.Unlock()
		for _, fn := range items {
			fn()
		}
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

//export onApplyBrightness
func onApplyBrightness() {
	if dash == nil {
		return
	}
	val := int(C.w_scale_get(dash.brightnessScale))
	writeBrightness(val)
	dash.UpdateBrightnessValue(val)
	if hassClient != nil {
		hassClient.PublishBrightnessToHass(dash.BrightnessPercent(val))
	}
}

//export onQuitClicked
func onQuitClicked() { C.gtk_main_quit() }

//export onSwipeStart
func onSwipeStart(x, y C.double) {
	markActivity()
	if dash == nil {
		return
	}
	dash.swipeStartX = float64(x)
	dash.swipeStartY = float64(y)
	dash.swipeActive = true
}

//export onSwipeEnd
func onSwipeEnd(x, y C.double) {
	if dash == nil || !dash.swipeActive {
		return
	}
	dash.swipeActive = false
	dx := float64(x) - dash.swipeStartX
	dy := float64(y) - dash.swipeStartY
	if absf(dx) < 90 || absf(dx) < absf(dy)+30 {
		return
	}
	if dx < 0 {
		dash.showView(dash.currentView + 1)
		return
	}
	dash.showView(dash.currentView - 1)
}

//export onToggleEntityClicked
func onToggleEntityClicked(entity *C.char) {
	markActivity()
	if hassClient == nil || entity == nil {
		return
	}
	hassClient.ToggleEntity(C.GoString(entity))
}

//export onMacroActionClicked
func onMacroActionClicked(action *C.char) {
	markActivity()
	if pcMacroClient == nil || action == nil {
		return
	}
	go pcMacroClient.Execute(C.GoString(action))
}

// ─── Dashboard ───
type DashboardOptions struct {
	HardwareLandscape bool
	HassLightEntities []string
	PCEnabled         bool
}

type Dashboard struct {
	window *C.GtkWidget

	options DashboardOptions

	// Clock panel
	greeting  *C.GtkWidget
	clockLbl  *C.GtkWidget
	dateLbl   *C.GtkWidget
	statusLbl *C.GtkWidget
	calMonth  *C.GtkWidget
	calYear   *C.GtkWidget
	calDays   [6][7]*C.GtkWidget

	// Devices
	brightnessScale  *C.GtkWidget
	brightnessVal    *C.GtkWidget
	brightnessMax    int
	batteryCache     string
	hassLightButtons map[string]*C.GtkWidget
	hassLightNames   map[string]string

	// Home Assistant cards
	mailUnread             *C.GtkWidget
	dashboardAgendaSummary *C.GtkWidget
	dashboardAgendaItems   [4]*C.GtkWidget
	calendarAgendaSummary  *C.GtkWidget
	calendarAgendaItems    [4]*C.GtkWidget
	pcConnStatus           *C.GtkWidget
	pcTrackTitle           *C.GtkWidget
	pcTrackArtist          *C.GtkWidget
	pcModeBtn              *C.GtkWidget
	pcMonitorBtn           *C.GtkWidget
	pcPlayPauseBtn         *C.GtkWidget

	// Views
	views             [viewCount]*C.GtkWidget
	indicators        [viewCount]*C.GtkWidget
	currentView       ViewID
	currentViewAtomic atomic.Int32

	// Now Playing persistent bar
	nowBar           *C.GtkWidget
	nowPlayingTrack  *C.GtkWidget
	nowPlayingArtist *C.GtkWidget
	nowPlayingStatus *C.GtkWidget

	// Info view status widgets
	infoConnStatus   *C.GtkWidget
	infoPCConnStatus *C.GtkWidget
	infoHassSummary  *C.GtkWidget
	infoBrightness   *C.GtkWidget

	swipeStartX float64
	swipeStartY float64
	swipeActive bool
}

func NewDashboard(options DashboardOptions) *Dashboard {
	stopKindleFramework()
	C.w_init()
	C.gtk_init(nil, nil)
	d := &Dashboard{options: options, brightnessMax: readMaxBrightness(), hassLightButtons: map[string]*C.GtkWidget{}, hassLightNames: map[string]string{}}
	if d.brightnessMax <= 0 {
		d.brightnessMax = 2399
	}
	d.window = C.w_win(C.gboolean(boolToInt(options.HardwareLandscape)))
	C.w_signal(d.window, C.CString("destroy"), C.GCallback(unsafe.Pointer(C.gtk_main_quit)))
	C.w_enable_swipe(d.window)

	root := C.w_vbox(0, 0)

	// View container
	vc := C.w_vbox(0, 0)
	d.views[ViewCalendar] = buildCalendarView(d)
	C.w_pack(vc, d.views[ViewCalendar], 1, 1, 0)
	d.views[ViewHome] = d.buildDashboardView()
	C.w_pack(vc, d.views[ViewHome], 1, 1, 0)
	d.views[ViewLauncher] = buildLauncherView(d)
	C.w_pack(vc, d.views[ViewLauncher], 1, 1, 0)
	d.views[ViewInfo] = d.buildInfoView()
	C.w_pack(vc, d.views[ViewInfo], 1, 1, 0)
	C.w_pack(root, vc, 1, 1, 0)

	// Indicators + Now Playing on same row
	bottomRow := C.w_hbox(0, 0)

	dr := C.w_hbox(0, 10)
	C.w_border(dr, 6)
	for i := 0; i < int(viewCount); i++ {
		dot := C.w_lbl()
		C.w_markup(dot, C.CString("<span font_desc='14'>●</span>"))
		C.w_fg(dot, C.CString("#d9d1bf"))
		d.indicators[i] = dot
		C.w_pack(dr, dot, 0, 0, 0)
	}
	C.w_pack(bottomRow, dr, 0, 0, 0)

	// ── Now Playing persistent bar ──
	nb := C.w_hbox(0, 4)
	C.w_border(nb, 4)
	d.nowPlayingTrack = C.w_lbl()
	C.w_markup(d.nowPlayingTrack, C.CString("<span font_desc='9' weight='bold' color='#626262'>Idle</span>"))
	C.w_align(d.nowPlayingTrack, 0, 0.5)
	C.w_pack(nb, d.nowPlayingTrack, 0, 0, 0)
	sep := C.w_lbl()
	C.w_markup(sep, C.CString("<span font_desc='9' color='#626262'> </span>"))
	C.w_pack(nb, sep, 0, 0, 0)
	d.nowPlayingArtist = C.w_lbl()
	C.w_markup(d.nowPlayingArtist, C.CString("<span font_desc='8' color='#626262'></span>"))
	C.w_align(d.nowPlayingArtist, 0, 0.5)
	C.w_pack(nb, d.nowPlayingArtist, 1, 1, 0)
	d.nowPlayingStatus = C.w_lbl()
	C.w_markup(d.nowPlayingStatus, C.CString("<span font_desc='8' color='#626262'>○</span>"))
	C.w_align(d.nowPlayingStatus, 1, 0.5)
	C.w_pack_end(nb, d.nowPlayingStatus, 0, 0, 0)
	d.nowBar = nb
	C.w_pack(bottomRow, nb, 1, 1, 0)

	// ── Media transport (persistent) ──
	const transportSize = 32
	tb := C.w_hbox(0, 4)
	C.w_border(tb, 4)
	C.w_pack(tb, d.newIconButton("prev_track", transportSize), 0, 0, 0)
	d.pcPlayPauseBtn = d.newIconButton("play_pause", transportSize)
	C.w_pack(tb, d.pcPlayPauseBtn, 0, 0, 0)
	C.w_pack(tb, d.newIconButton("next_track", transportSize), 0, 0, 0)
	C.w_pack(bottomRow, tb, 0, 0, 0)

	C.w_pack(root, bottomRow, 0, 0, 0)

	C.w_add(d.window, root)

	d.showView(ViewHome)
	return d
}

func (d *Dashboard) Show() {
	C.w_show_all(d.window)
	// gtk_widget_show_all() re-shows children hidden during construction, so
	// re-apply the selected view after realizing/showing the tree.
	d.showView(d.currentView)
	if !d.options.HardwareLandscape {
		C.w_override()
	}
}

// Redraw forces a full repaint of the window. Used after resuming from
// suspend, where the e-ink/framebuffer state may be stale.
func (d *Dashboard) Redraw() {
	d.runOnUI(func() {
		C.w_redraw(d.window)
	})
}

func (d *Dashboard) runOnUI(fn func()) {
	if d == nil {
		return
	}
	enqueueUI(fn)
}

func (d *Dashboard) Loop() { C.gtk_main() }

func (d *Dashboard) CurrentView() ViewID {
	if d == nil {
		return ViewHome
	}
	return ViewID(d.currentViewAtomic.Load())
}

// ── Dashboard main view ──
func (d *Dashboard) buildDashboardView() *C.GtkWidget {
	vb := C.w_vbox(0, 6)
	C.w_border(vb, 10)

	greetingFont := "10"
	clockFont := "66"
	dateFont := "16"
	statusFont := "10"
	calMonthFont := "15"
	calYearFont := "11"
	calDayFont := "9"
	clockPad := uint(12)
	calendarMiniWidth := 170
	if d.options.HardwareLandscape {
		greetingFont = "8"
		clockFont = "48"
		dateFont = "14"
		statusFont = "9"
		calMonthFont = "14"
		calYearFont = "11"
		calDayFont = "10"
		clockPad = 6
		calendarMiniWidth = 240
	}

	// ── Clock panel ──
	cf := C.w_frame(nil)
	C.w_shadow(cf, C.GTK_SHADOW_ETCHED_IN)
	ch := C.w_hbox(0, 12)
	C.w_border(ch, C.uint(clockPad))

	// Left column
	left := C.w_vbox(0, 0)
	d.greeting = C.w_lbl()
	C.w_markup(d.greeting, C.CString(fmt.Sprintf("<span font_desc='%s' weight='bold' color='#626262'>GOOD DAY</span>", greetingFont)))
	C.w_align(d.greeting, 0, 0.5)
	C.w_pack(left, d.greeting, 0, 0, 0)

	d.clockLbl = C.w_lbl()
	C.w_markup(d.clockLbl, C.CString(fmt.Sprintf("<span font_desc='%s' weight='950'>--:--</span>", clockFont)))
	C.w_align(d.clockLbl, 0, 0.5)
	C.w_pack(left, d.clockLbl, 0, 0, 0)

	d.dateLbl = C.w_lbl()
	C.w_markup(d.dateLbl, C.CString(fmt.Sprintf("<span font_desc='%s' weight='850'>Loading...</span>", dateFont)))
	C.w_align(d.dateLbl, 0, 0.5)
	C.w_pack(left, d.dateLbl, 0, 0, 0)

	// Status line: Bat + Mail
	sl := C.w_hbox(0, 10)
	d.statusLbl = C.w_lbl()
	C.w_markup(d.statusLbl, C.CString(fmt.Sprintf("<span font_desc='%s' color='#626262'>Bat --%%</span>", statusFont)))
	C.w_pack(sl, d.statusLbl, 0, 0, 0)

	d.mailUnread = C.w_lbl()
	C.w_markup(d.mailUnread, C.CString(fmt.Sprintf("<span font_desc='%s' color='#626262'>✉ 0</span>", statusFont)))
	C.w_pack(sl, d.mailUnread, 0, 0, 0)
	C.w_pack(left, sl, 0, 0, 0)

	C.w_pack(ch, left, 1, 1, 0)

	// Right: calendar mini
	right := C.w_vbox(0, 2)
	C.w_border(right, 4)
	C.w_size(right, C.int(calendarMiniWidth), -1)

	mr := C.w_hbox(0, 0)
	d.calMonth = C.w_lbl()
	C.w_markup(d.calMonth, C.CString(fmt.Sprintf("<span font_desc='%s' weight='950'>Month</span>", calMonthFont)))
	C.w_align(d.calMonth, 0, 0.5)
	C.w_pack(mr, d.calMonth, 0, 0, 4)

	d.calYear = C.w_lbl()
	C.w_markup(d.calYear, C.CString(fmt.Sprintf("<span font_desc='%s' weight='bold' color='#626262'>----</span>", calYearFont)))
	C.w_align(d.calYear, 1, 0.5)
	C.w_pack_end(mr, d.calYear, 1, 1, 0)
	C.w_pack(right, mr, 0, 0, 0)

	grid := C.w_table(6, 7)
	C.w_size(grid, C.int(calendarMiniWidth), -1)
	C.w_table_spacing(grid, 1, 2)
	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			cell := C.w_lbl()
			C.w_markup(cell, C.CString(fmt.Sprintf("<span font_desc='%s'> </span>", calDayFont)))
			C.w_align(cell, 0.5, 0.5)
			d.calDays[r][c] = cell
			C.w_table_put(grid, cell, C.int(c), C.int(c+1), C.int(r), C.int(r+1))
		}
	}
	C.w_pack(right, grid, 1, 1, 0)
	C.w_pack(ch, right, 0, 1, 0)
	C.w_add(cf, ch)
	C.w_pack(vb, cf, 0, 0, 0)

	// ── Bottom row: cards ──
	btm := C.w_hbox(0, 6)

	// Devices
	// Devices — two big square buttons, one per row
	dc := frameCard("Devices")
	numLights := len(d.options.HassLightEntities)
	devGrid := C.w_table(C.int(numLights), 1)
	C.w_table_spacing(devGrid, 8, 8)
	C.w_border(devGrid, 8)

	for i, entity := range d.options.HassLightEntities {
		name := prettyEntityName(entity)
		btnText := lightButtonLabel(name, "--")
		cs := C.CString(btnText)
		btnName := C.CString(btnNameToggle)
		btn := C.w_btn_named(cs, btnName)
		C.free(unsafe.Pointer(btnName))
		C.free(unsafe.Pointer(cs))
		es := C.CString(entity)
		C.w_bind_toggle(btn, es)
		C.free(unsafe.Pointer(es))
		C.w_table_put(devGrid, btn, 0, 1, C.int(i), C.int(i+1))
		d.hassLightButtons[entity] = btn
		d.hassLightNames[entity] = name
	}

	C.w_add(dc, devGrid)
	C.w_size(dc, 220, -1)
	C.w_pack(btm, dc, 0, 1, 0)

	// Middle col: Agenda
	mc := C.w_vbox(0, 8)
	C.w_pack(mc, d.agendaCard(), 1, 1, 0)
	C.w_size(mc, 260, -1)
	C.w_pack(btm, mc, 1, 1, 0)

	// (Right col removed — music now lives in persistent bottom bar)

	C.w_pack(vb, btm, 1, 1, 0)
	return vb
}

func (d *Dashboard) mailCard() *C.GtkWidget { return nil } // Removed

func (d *Dashboard) agendaCard() *C.GtkWidget {
	f := frameCard("Agenda")
	vb := C.w_vbox(0, 6)
	C.w_border(vb, 10)
	d.dashboardAgendaSummary = C.w_lbl()
	C.w_pack(vb, d.dashboardAgendaSummary, 0, 0, 0)
	for i := range d.dashboardAgendaItems {
		d.dashboardAgendaItems[i] = C.w_lbl()
		C.w_align(d.dashboardAgendaItems[i], 0, 0.5)
		C.w_pack(vb, d.dashboardAgendaItems[i], 0, 0, 0)
	}
	C.w_add(f, vb)
	d.UpdateAgenda(AgendaData{Summary: "No upcoming events"})
	return f
}

func (d *Dashboard) showView(idx ViewID) {
	markActivity()
	if idx < 0 {
		idx = ViewCalendar
	}
	if idx >= viewCount {
		idx = ViewInfo
	}
	// PC Macro: stop streaming when leaving the launcher view,
	// start on demand when entering it.
	prevIdx := d.currentView
	for i, v := range d.views {
		viewID := ViewID(i)
		if viewID == idx {
			C.w_show(v)
			setMarkup(d.indicators[i], "<span font_desc='14'>●</span>")
			setFG(d.indicators[i], "#252525")
		} else {
			C.w_hide(v)
			setMarkup(d.indicators[i], "<span font_desc='14'>●</span>")
			setFG(d.indicators[i], "#d9d1bf")
		}
	}
	d.currentView = idx
	d.currentViewAtomic.Store(int32(idx))

	// Open the SSE stream on entering the launcher view, close it on leaving.
	if pcMacroClient != nil {
		if idx == ViewLauncher && prevIdx != ViewLauncher {
			pcMacroClient.Touch()
		} else if prevIdx == ViewLauncher && idx != ViewLauncher {
			pcMacroClient.StopStreaming()
		}
	}

	// View changes are a user-visible screen transition. Refresh the visible
	// view immediately instead of waiting for the next minute tick / poll cycle.
	d.refreshVisibleViewOnUI(time.Now())
}

func (d *Dashboard) UpdateClock(now time.Time) {
	d.runOnUI(func() {
		d.updateClock(now)
	})
}

func (d *Dashboard) RefreshVisibleView(now time.Time) {
	d.runOnUI(func() {
		d.refreshVisibleViewOnUI(now)
	})
}

func (d *Dashboard) refreshVisibleViewOnUI(now time.Time) {
	if d.currentView == ViewHome {
		d.updateClock(now)
	}
	C.w_redraw(d.window)
}

func (d *Dashboard) updateClock(now time.Time) {
	greetingFont := "10"
	clockFont := "66"
	dateFont := "16"
	statusFont := "10"
	calMonthFont := "15"
	calYearFont := "11"
	calDayFont := "9"
	if d.options.HardwareLandscape {
		greetingFont = "8"
		clockFont = "48"
		dateFont = "14"
		statusFont = "9"
		calMonthFont = "14"
		calYearFont = "11"
		calDayFont = "10"
	}

	h := now.Hour()
	greet := "Good night"
	switch {
	case h >= 5 && h < 12:
		greet = "Good morning"
	case h >= 12 && h < 17:
		greet = "Good afternoon"
	case h >= 17 && h < 22:
		greet = "Good evening"
	}

	months := [13]string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	ym, mm, dm := now.Date()
	first := time.Date(ym, mm, 1, 0, 0, 0, 0, time.UTC)
	sdow := int(first.Weekday())
	dim := time.Date(ym, mm+1, 0, 0, 0, 0, 0, time.UTC).Day()

	setMarkup(d.greeting, fmt.Sprintf("<span font_desc='%s' weight='bold' color='#626262'>%s</span>", greetingFont, greet))
	setMarkup(d.clockLbl, fmt.Sprintf("<span font_desc='%s' weight='950'>%s</span>", clockFont, now.Format("15:04")))
	setMarkup(d.dateLbl, fmt.Sprintf("<span font_desc='%s' weight='850'>%s</span>", dateFont, now.Format("Monday, January 2")))
	// Battery is now updated by the dedicated battery polling goroutine.
	// d.batteryCache is set via dash.UpdateBattery() called from main.go.
	setMarkup(d.statusLbl, fmt.Sprintf("<span font_desc='%s' color='#626262'>Bat %s%%</span>", statusFont, d.batteryCache))
	setMarkup(d.calMonth, fmt.Sprintf("<span font_desc='%s' weight='950'>%s</span>", calMonthFont, months[mm]))
	setMarkup(d.calYear, fmt.Sprintf("<span font_desc='%s' weight='bold' color='#626262'> %d</span>", calYearFont, ym))

	day := 1
	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			cell := d.calDays[r][c]
			if (r == 0 && c < sdow) || day > dim {
				setMarkup(cell, fmt.Sprintf("<span font_desc='%s'> </span>", calDayFont))
			} else {
				m := fmt.Sprintf("<span font_desc='%s'>%d</span>", calDayFont, day)
				if day == dm {
					m = fmt.Sprintf("<span font_desc='%s' weight='bold' foreground='#ffffff' background='#252525'>%d</span>", calDayFont, day)
				}
				setMarkup(cell, m)
				day++
			}
		}
	}
}

// ── View helpers ──
func buildCalendarView(d *Dashboard) *C.GtkWidget {
	vb := C.w_vbox(0, 6)
	C.w_border(vb, 10)
	card := frameCard("Upcoming Agenda")

	// Agenda Items
	vbA := C.w_vbox(0, 4)
	C.w_border(vbA, 6)

	// Add both to a container inside the frame
	sub := C.w_lbl()
	C.w_markup(sub, C.CString("<span font_desc='11' color='#626262'>Next 7 Days</span>"))
	C.w_align(sub, 0, 0.5)
	C.w_pack(vbA, sub, 0, 0, 0)

	d.calendarAgendaSummary = C.w_lbl()
	C.w_align(d.calendarAgendaSummary, 0, 0.5)
	C.w_pack(vbA, d.calendarAgendaSummary, 0, 0, 0)
	for i := range d.calendarAgendaItems {
		d.calendarAgendaItems[i] = C.w_lbl()
		C.w_align(d.calendarAgendaItems[i], 0, 0.5)
		C.w_pack(vbA, d.calendarAgendaItems[i], 0, 0, 0)
	}
	C.w_add(card, vbA)
	C.w_pack(vb, card, 1, 1, 0)

	d.UpdateAgenda(AgendaData{Summary: "Loading calendar..."})
	return vb
}

func buildLauncherView(d *Dashboard) *C.GtkWidget {
	vb := C.w_vbox(0, 0)
	C.w_border(vb, 10)

	// ── 3×3 grid of 50×50 icon buttons that grows to fill available X and Y ──
	const iconSize = 100
	grid := C.w_table(3, 3)
	C.w_table_spacing(grid, 10, 10)

	buttons := []struct {
		action string
		row    int
		col    int
		assign func(*C.GtkWidget)
	}{
		{action: "pc_mode_toggle", row: 0, col: 0, assign: func(btn *C.GtkWidget) { d.pcModeBtn = btn }},
		{action: "mute_mic", row: 0, col: 1},
		{action: "monitor_toggle", row: 0, col: 2, assign: func(btn *C.GtkWidget) { d.pcMonitorBtn = btn }},
		{action: "launch_chrome", row: 1, col: 0},
		{action: "launch_mail", row: 1, col: 1},
		{action: "sleep", row: 1, col: 2},
		{action: "restart", row: 2, col: 0},
		{action: "launch_fortnite", row: 2, col: 1},
		{action: "shutdown", row: 2, col: 2},
	}
	for _, spec := range buttons {
		btn := d.newIconButton(spec.action, iconSize)
		if spec.assign != nil {
			spec.assign(btn)
		}
		C.w_table_put_center(grid, btn, C.int(spec.col), C.int(spec.col+1), C.int(spec.row), C.int(spec.row+1))
	}

	C.w_pack(vb, grid, 1, 1, 0)
	d.SetPCConnectionStatus(map[bool]string{true: "Disconnected", false: "Not configured"}[d.options.PCEnabled])
	d.UpdatePCStatus(PCStatus{Status: "Idle"})
	return vb
}

func (d *Dashboard) buildInfoView() *C.GtkWidget {
	vb := C.w_vbox(0, 6)
	C.w_border(vb, 10)

	// System status
	card1 := frameCard("System")
	vb1 := C.w_vbox(0, 6)
	C.w_border(vb1, 10)

	d.infoConnStatus = C.w_lbl()
	C.w_markup(d.infoConnStatus, C.CString("<span font_desc='10' weight='bold'>Home Assistant</span>"))
	C.w_pack(vb1, d.infoConnStatus, 0, 0, 0)
	d.infoHassSummary = C.w_lbl()
	C.w_markup(d.infoHassSummary, C.CString("<span font_desc='10' color='#626262'>Disconnected</span>"))
	C.w_pack(vb1, d.infoHassSummary, 0, 0, 0)

	d.infoPCConnStatus = C.w_lbl()
	label := "Not configured"
	if d.options.PCEnabled {
		label = "Disconnected"
	}
	C.w_markup(d.infoPCConnStatus, C.CString(fmt.Sprintf("<span font_desc='10' color='#626262'>PC Macro: %s</span>", label)))
	C.w_pack(vb1, d.infoPCConnStatus, 0, 0, 0)

	C.w_add(card1, vb1)
	C.w_pack(vb, card1, 0, 0, 0)

	// Brightness card
	card2 := frameCard("Brightness")
	vb2 := C.w_vbox(0, 6)
	C.w_border(vb2, 10)

	d.brightnessScale = C.w_hscale(0, C.double(d.brightnessMax), 1)
	C.w_size(d.brightnessScale, -1, -1)
	C.w_pack(vb2, d.brightnessScale, 1, 1, 0)
	d.brightnessVal = C.w_lbl()
	C.w_markup(d.brightnessVal, C.CString("<span font_desc='9' color='#626262'>Brightness --%</span>"))
	C.w_pack(vb2, d.brightnessVal, 0, 0, 0)
	d.UpdateBrightnessValue(readBrightness())

	applyBtnName := C.CString(btnNameApply)
	ab := C.w_btn_named(C.CString("Apply"), applyBtnName)
	C.free(unsafe.Pointer(applyBtnName))
	C.w_signal(ab, C.CString("clicked"), C.GCallback(unsafe.Pointer(C.onApplyBrightness)))
	C.w_pack(vb2, ab, 0, 0, 0)

	C.w_add(card2, vb2)
	C.w_pack(vb, card2, 0, 0, 0)

	// Version info
	card3 := frameCard("Kindle Dashboard")
	vb3 := C.w_vbox(0, 6)
	C.w_border(vb3, 10)
	v := C.w_lbl()
	C.w_markup(v, C.CString("<span font_desc='9' color='#626262'>Native GTK+2 • 800x600</span>"))
	C.w_pack(vb3, v, 0, 0, 0)
	C.w_add(card3, vb3)
	C.w_pack(vb, card3, 0, 0, 0)

	// Push remaining space
	spacer := C.w_lbl()
	C.w_pack(vb, spacer, 1, 1, 0)

	return vb
}

func (d *Dashboard) newMacroButton(label, action string, width int) *C.GtkWidget {
	mediaBtnName := C.CString(btnNameMedia)
	cs := C.CString(label)
	btn := C.w_btn_named(cs, mediaBtnName)
	C.free(unsafe.Pointer(mediaBtnName))
	C.free(unsafe.Pointer(cs))
	// Add icon based on action name
	iconCS := C.CString(action)
	C.w_btn_set_icon(btn, iconCS)
	C.free(unsafe.Pointer(iconCS))
	as := C.CString(action)
	C.w_bind_macro(btn, as)
	C.free(unsafe.Pointer(as))
	if width > 0 {
		C.w_size(btn, C.int(width), C.int(width)) // square
	}
	return btn
}

// newIconButton creates an icon-only macro button, fixed at size×size.
func (d *Dashboard) newIconButton(action string, size int) *C.GtkWidget {
	mediaBtnName := C.CString(btnNameMedia)
	iconCS := C.CString(action)
	btn := C.w_btn_icon(iconCS, mediaBtnName)
	C.free(unsafe.Pointer(mediaBtnName))
	C.free(unsafe.Pointer(iconCS))
	as := C.CString(action)
	C.w_bind_macro(btn, as)
	C.free(unsafe.Pointer(as))
	if size > 0 {
		C.w_size(btn, C.int(size), C.int(size))
	}
	return btn
}

func frameCard(title string) *C.GtkWidget {
	f := C.w_frame(C.CString(title))
	C.w_shadow(f, C.GTK_SHADOW_ETCHED_IN)
	return f
}

func (d *Dashboard) SetConnectionStatus(status string) {
	d.runOnUI(func() {
		if d.infoHassSummary == nil {
			return
		}
		setMarkup(d.infoHassSummary, fmt.Sprintf("<span font_desc='10' color='#626262'>%s</span>", esc(status)))
	})
}

func (d *Dashboard) SetPCConnectionStatus(status string) {
	d.runOnUI(func() {
		if d.infoPCConnStatus == nil {
			return
		}
		setMarkup(d.infoPCConnStatus, fmt.Sprintf("<span font_desc='10' color='#626262'>PC Macro: %s</span>", esc(status)))
	})
}

func (d *Dashboard) UpdatePCStatus(status PCStatus) {
	d.runOnUI(func() {
		if d.pcTrackTitle != nil {
			track := fallback(status.Track, "Not Playing")
			setMarkup(d.pcTrackTitle, fmt.Sprintf("<span font_desc='11' weight='bold'>%s</span>", esc(shorten(track, 34))))
		}
		if d.pcTrackArtist != nil {
			artist := fallback(status.Artist, map[bool]string{true: "PC Idle", false: "PC"}[status.Status == "Idle"])
			setMarkup(d.pcTrackArtist, fmt.Sprintf("<span font_desc='9' color='#626262'>%s</span>", esc(shorten(artist, 40))))
		}
		if d.pcModeBtn != nil {
			icon := "pc_mode_save"
			if strings.EqualFold(status.GamingMode, "power") {
				icon = "pc_mode_power"
			}
			setButtonIcon(d.pcModeBtn, icon)
		}
		if d.pcMonitorBtn != nil {
			icon := "monitor_off"
			if status.MonitorOn {
				icon = "monitor_on"
			}
			setButtonIcon(d.pcMonitorBtn, icon)
		}
		if d.pcPlayPauseBtn != nil {
			if strings.EqualFold(status.Status, "playing") {
				setButtonIcon(d.pcPlayPauseBtn, "pause")
			} else {
				setButtonIcon(d.pcPlayPauseBtn, "play_pause")
			}
		}
		// Update now-playing bar
		d.updateNowPlayingFromPC(status)
	})
}

func (d *Dashboard) updateNowPlayingFromPC(status PCStatus) {
	track := fallback(status.Track, "")
	artist := fallback(status.Artist, "")
	stat := fallback(status.Status, "Idle")
	if track == "" && artist == "" {
		setMarkup(d.nowPlayingTrack, fmt.Sprintf("<span font_desc='9' weight='bold' color='#626262'>PC Idle</span>"))
		setMarkup(d.nowPlayingArtist, fmt.Sprintf("<span font_desc='9' color='#626262'>%s</span>", esc(stat)))
		setMarkup(d.nowPlayingStatus, fmt.Sprintf("<span font_desc='8' color='#626262'>○</span>"))
	} else {
		setMarkup(d.nowPlayingTrack, fmt.Sprintf("<span font_desc='9' weight='bold'>%s</span>", esc(shorten(track, 60))))
		setMarkup(d.nowPlayingArtist, fmt.Sprintf("<span font_desc='9' color='#626262'>%s</span>", esc(shorten(artist, 50))))
		badge := ">"
		if strings.EqualFold(stat, "paused") {
			badge = "|"
		}
		setMarkup(d.nowPlayingStatus, fmt.Sprintf("<span font_desc='8' weight='bold' color='#626262'>%s</span>", esc(badge)))
	}
}

func (d *Dashboard) UpdateMusic(m MusicData) {
	d.runOnUI(func() {
		// Update now-playing bar (music card was removed)
		d.updateNowPlaying(m)
	})
}

func (d *Dashboard) updateNowPlaying(m MusicData) {
	track := m.Track
	artist := m.Artist
	if artist == "" {
		artist = m.Album
	}
	if artist == "" {
		artist = m.Source
	}
	if track == "" || track == "No track" {
		setMarkup(d.nowPlayingTrack, fmt.Sprintf("<span font_desc='9' weight='bold' color='#626262'>No music</span>"))
		setMarkup(d.nowPlayingArtist, fmt.Sprintf("<span font_desc='9' color='#626262'>%s</span>", esc(m.Device)))
		setMarkup(d.nowPlayingStatus, fmt.Sprintf("<span font_desc='8' color='#626262'>○</span>"))
	} else {
		badge := ">"
		if strings.EqualFold(m.State, "paused") {
			badge = "|"
		}
		setMarkup(d.nowPlayingTrack, fmt.Sprintf("<span font_desc='9' weight='bold'>%s</span>", esc(shorten(track, 60))))
		setMarkup(d.nowPlayingArtist, fmt.Sprintf("<span font_desc='9' color='#626262'>%s</span>", esc(shorten(artist, 50))))
		setMarkup(d.nowPlayingStatus, fmt.Sprintf("<span font_desc='8' weight='bold' color='#626262'>%s</span>", esc(badge)))
	}
}

func (d *Dashboard) UpdateMail(m MailData) {
	d.runOnUI(func() {
		if d.mailUnread == nil {
			return
		}
		setMarkup(d.mailUnread, fmt.Sprintf("<span font_desc='11' color='#626262'>✉ %d</span>", m.Unread))
	})
}

func (d *Dashboard) UpdateLight(light LightData) {
	d.runOnUI(func() {
		btn := d.hassLightButtons[light.EntityID]
		if btn == nil {
			return
		}
		name := light.Name
		if name == "" {
			name = d.hassLightNames[light.EntityID]
		}
		if name == "" {
			name = prettyEntityName(light.EntityID)
		}
		d.hassLightNames[light.EntityID] = name
		setButtonMarkup(btn, lightButtonLabel(name, light.State))
		// Set button appearance based on state
		on := strings.EqualFold(light.State, "on") || strings.EqualFold(light.State, "open")
		if on {
			C.w_btn_bg(btn, C.CString("#252525"))
			// White text in all states (no hover effect)
			C.w_set_btn_label_fg(btn, C.CString("#ffffff"))
		} else {
			C.w_btn_bg(btn, C.CString("#f0f0f0"))
			// Dark text normally, white only when pressed (ACTIVE)
			C.w_set_btn_label_fg(btn, C.CString("#000000"))
			C.w_set_btn_label_fg_active_white(btn)
		}
	})
}

func (d *Dashboard) UpdateAgenda(a AgendaData) {
	d.runOnUI(func() {
		d.updateAgendaWidgets(d.dashboardAgendaSummary, d.dashboardAgendaItems[:], a, 32, 38)
		d.updateAgendaWidgets(d.calendarAgendaSummary, d.calendarAgendaItems[:], a, 54, 70)
	})
}

func (d *Dashboard) updateAgendaWidgets(summaryLabel *C.GtkWidget, itemLabels []*C.GtkWidget, a AgendaData, summaryMax, itemMax int) {
	if summaryLabel == nil {
		return
	}
	summary := a.Summary
	if summary == "" {
		summary = "No events"
	}
	setMarkup(summaryLabel, fmt.Sprintf("<span font_desc='10' color='#626262'>%s</span>", esc(shorten(summary, summaryMax))))
	for i, lbl := range itemLabels {
		text := ""
		if i < len(a.Events) {
			e := a.Events[i]
			if e.Time != "" {
				text = e.Time + "  " + e.Title
			} else {
				text = e.Title
			}
		}
		setMarkup(lbl, fmt.Sprintf("<span font_desc='9'>%s</span>", esc(shorten(text, itemMax))))
	}
}

func (d *Dashboard) UpdateBrightnessValue(val int) {
	d.runOnUI(func() {
		if d.brightnessMax <= 0 {
			d.brightnessMax = 2399
		}
		if val < 0 {
			val = 0
		}
		if val > d.brightnessMax {
			val = d.brightnessMax
		}
		if d.brightnessScale != nil {
			C.w_scale_set(d.brightnessScale, C.double(val))
		}
		if d.brightnessVal != nil {
			setMarkup(d.brightnessVal, fmt.Sprintf("<span font_desc='9' color='#626262'>Brightness %d%%</span>", d.BrightnessPercent(val)))
		}
	})
}

func (d *Dashboard) BrightnessPercent(val int) int {
	if d.brightnessMax <= 0 {
		return 0
	}
	return (val*100 + d.brightnessMax/2) / d.brightnessMax
}

func (d *Dashboard) UpdateBattery(capacity string) {
	d.runOnUI(func() {
		d.batteryCache = capacity
		statusFont := "10"
		if d.options.HardwareLandscape {
			statusFont = "9"
		}
		setMarkup(d.statusLbl, fmt.Sprintf("<span font_desc='%s' color='#626262'>Bat %s%%</span>", statusFont, d.batteryCache))
	})
}

func (d *Dashboard) SetBrightnessPercent(percent int) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	val := percent * d.brightnessMax / 100
	writeBrightness(val)
	d.UpdateBrightnessValue(val)
}

func setMarkup(label *C.GtkWidget, markup string) {
	cs := C.CString(markup)
	defer C.free(unsafe.Pointer(cs))
	C.w_markup(label, cs)
}

func setFG(widget *C.GtkWidget, color string) {
	cs := C.CString(color)
	defer C.free(unsafe.Pointer(cs))
	C.w_fg(widget, cs)
}

func setButtonLabel(button *C.GtkWidget, text string) {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	C.w_btn_text(button, cs)
}

func setButtonIcon(button *C.GtkWidget, icon string) {
	cs := C.CString(icon)
	defer C.free(unsafe.Pointer(cs))
	C.w_btn_set_icon(button, cs)
}

func setButtonMarkup(button *C.GtkWidget, markup string) {
	cs := C.CString(markup)
	defer C.free(unsafe.Pointer(cs))
	C.w_btn_markup(button, cs)
}
