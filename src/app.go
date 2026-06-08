package main

/*
#cgo LDFLAGS: -lgtk-x11-2.0 -lgdk-x11-2.0 -lgdk_pixbuf-2.0 -lpangocairo-1.0 -lpango-1.0 -lcairo -latk-1.0 -lgio-2.0 -lgobject-2.0 -lglib-2.0 -lX11 -lXext -lXrender
#cgo CFLAGS:

#include <gtk/gtk.h>
#include <gdk/gdk.h>
#include <X11/Xlib.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

extern void onApplyBrightness();
extern void onQuitClicked();

// Widget factories
// Call this before any GTK/GLib usage for GTK 2.10 compat
static void w_init() { g_type_init(); }

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
static GtkWidget* w_hscale(double min, double max, double step) {
	GtkObject *a = gtk_adjustment_new(min, min, max, step, step*10, 0);
	return gtk_hscale_new(GTK_ADJUSTMENT(a));
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
static void w_shadow(GtkWidget *w, GtkShadowType s) { gtk_frame_set_shadow_type(GTK_FRAME(w), s); }
static void w_table_put(GtkWidget *t, GtkWidget *w, gint l, gint r, gint tp, gint bt) {
	gtk_table_attach(GTK_TABLE(t), w, l, r, tp, bt, GTK_EXPAND|GTK_FILL, GTK_EXPAND|GTK_FILL, 0, 0);
}
static void w_table_spacing(GtkWidget *t, guint rs, guint cs) {
	gtk_table_set_row_spacings(GTK_TABLE(t), rs);
	gtk_table_set_col_spacings(GTK_TABLE(t), cs);
}
static void w_fg(GtkWidget *w, const char *c) {
	GdkColor col; gdk_color_parse(c, &col);
	gtk_widget_modify_fg(w, GTK_STATE_NORMAL, &col);
}
static void w_bg(GtkWidget *w, const char *c) {
	GdkColor col; gdk_color_parse(c, &col);
	gtk_widget_modify_bg(w, GTK_STATE_NORMAL, &col);
}
static double w_scale_get(GtkWidget *s) { return gtk_range_get_value(GTK_RANGE(s)); }
static void w_signal(GtkWidget *w, const char *s, GCallback cb) {
	g_signal_connect(G_OBJECT(w), s, cb, NULL);
}
static void w_show(GtkWidget *w)   { gtk_widget_show(w); }
static void w_hide(GtkWidget *w)   { gtk_widget_hide(w); }
static void w_show_all(GtkWidget *w) { gtk_widget_show_all(w); }

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
	"time"
	"unsafe"
)

// ─── State ───
var dash *Dashboard

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

//export onApplyBrightness
func onApplyBrightness() {
	if dash == nil {
		return
	}
	val := int(C.w_scale_get(dash.brightnessScale))
	writeBrightness(val)
}

//export onQuitClicked
func onQuitClicked() { C.gtk_main_quit() }

// ─── Dashboard ───
type DashboardOptions struct {
	HardwareLandscape bool
}

type Dashboard struct {
	window *C.GtkWidget

	options DashboardOptions

	// Clock panel
	greeting *C.GtkWidget
	clockLbl *C.GtkWidget
	dateLbl  *C.GtkWidget
	calMonth *C.GtkWidget
	calYear  *C.GtkWidget
	calDays  [6][7]*C.GtkWidget

	// Devices
	brightnessScale *C.GtkWidget
	brightnessVal   *C.GtkWidget
	batteryVal      *C.GtkWidget

	// Views
	views       [3]*C.GtkWidget
	indicators  [3]*C.GtkWidget
	currentView int
}

func NewDashboard(options DashboardOptions) *Dashboard {
	C.w_init()
	C.gtk_init(nil, nil)
	d := &Dashboard{options: options}
	d.window = C.w_win(C.gboolean(boolToInt(options.HardwareLandscape)))
	C.w_signal(d.window, C.CString("destroy"), C.GCallback(unsafe.Pointer(C.gtk_main_quit)))

	root := C.w_vbox(0, 0)

	// View container
	vc := C.w_vbox(0, 0)
	d.views[0] = buildCalendarView()
	C.w_pack(vc, d.views[0], 1, 1, 0)
	d.views[1] = d.buildDashboardView()
	C.w_pack(vc, d.views[1], 1, 1, 0)
	d.views[2] = buildLauncherView()
	C.w_pack(vc, d.views[2], 1, 1, 0)
	C.w_pack(root, vc, 1, 1, 0)

	// Indicators
	dr := C.w_hbox(0, 6)
	C.w_border(dr, 4)
	for i := 0; i < 3; i++ {
		dot := C.w_lbl()
		C.w_markup(dot, C.CString("<span font_desc='14'>●</span>"))
		C.w_fg(dot, C.CString("#d9d1bf"))
		d.indicators[i] = dot
		C.w_pack(dr, dot, 0, 0, 0)
	}
	C.w_pack(root, dr, 0, 0, 0)
	C.w_add(d.window, root)

	d.showView(1)
	return d
}

func (d *Dashboard) Show() {
	C.w_show_all(d.window)
	if !d.options.HardwareLandscape {
		C.w_override()
	}
}

func (d *Dashboard) Loop() { C.gtk_main() }

// ── Dashboard main view ──
func (d *Dashboard) buildDashboardView() *C.GtkWidget {
	vb := C.w_vbox(0, 6)
	C.w_border(vb, 10)

	// ── Clock panel ──
	cf := C.w_frame(nil)
	C.w_shadow(cf, C.GTK_SHADOW_ETCHED_IN)
	ch := C.w_hbox(0, 12)
	C.w_border(ch, 12)

	// Left column
	left := C.w_vbox(0, 0)
	d.greeting = C.w_lbl()
	C.w_markup(d.greeting, C.CString("<span font_desc='10' weight='bold' color='#626262'>GOOD DAY</span>"))
	C.w_align(d.greeting, 0, 0.5)
	C.w_pack(left, d.greeting, 0, 0, 0)

	d.clockLbl = C.w_lbl()
	C.w_markup(d.clockLbl, C.CString("<span font_desc='72' weight='950'>--:--</span>"))
	C.w_align(d.clockLbl, 0, 0.5)
	C.w_pack(left, d.clockLbl, 0, 0, 0)

	d.dateLbl = C.w_lbl()
	C.w_markup(d.dateLbl, C.CString("<span font_desc='18' weight='850'>Loading...</span>"))
	C.w_align(d.dateLbl, 0, 0.5)
	C.w_pack(left, d.dateLbl, 0, 0, 0)

	statusL := C.w_lbl()
	C.w_markup(statusL, C.CString("<span font_desc='11' color='#626262'>Kindle home • local dashboard</span>"))
	C.w_align(statusL, 0, 0.5)
	C.w_pack(left, statusL, 0, 0, 0)

	C.w_pack(ch, left, 1, 1, 0)

	// Right: calendar mini
	right := C.w_vbox(0, 2)
	C.w_border(right, 4)

	mr := C.w_hbox(0, 0)
	ck := C.w_lbl()
	C.w_markup(ck, C.CString("<span font_desc='9' weight='bold' color='#626262'>CALENDAR</span>"))
	C.w_align(ck, 0, 0.5)
	C.w_pack(mr, ck, 0, 0, 0)

	d.calMonth = C.w_lbl()
	C.w_markup(d.calMonth, C.CString("<span font_desc='15' weight='950'>Month</span>"))
	C.w_align(d.calMonth, 0, 0.5)
	C.w_pack(mr, d.calMonth, 0, 0, 4)

	d.calYear = C.w_lbl()
	C.w_markup(d.calYear, C.CString("<span font_desc='11' weight='bold' color='#626262'>----</span>"))
	C.w_align(d.calYear, 1, 0.5)
	C.w_pack_end(mr, d.calYear, 1, 1, 0)
	C.w_pack(right, mr, 0, 0, 0)

	grid := C.w_table(6, 7)
	C.w_table_spacing(grid, 1, 2)
	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			cell := C.w_lbl()
			C.w_markup(cell, C.CString("<span font_desc='9'> </span>"))
			C.w_align(cell, 0.5, 0.5)
			d.calDays[r][c] = cell
			C.w_table_put(grid, cell, C.int(c), C.int(c+1), C.int(r), C.int(r+1))
		}
	}
	C.w_pack(right, grid, 1, 1, 0)
	C.w_pack(ch, right, 0, 0, 0)
	C.w_add(cf, ch)
	C.w_pack(vb, cf, 0, 0, 0)

	// ── Bottom row: cards ──
	btm := C.w_hbox(0, 6)

	// Devices
	dc := frameCard("Devices")
	vbd := C.w_vbox(0, 4)
	C.w_border(vbd, 6)
	sub := C.w_lbl()
	C.w_markup(sub, C.CString("<span font_desc='10' color='#626262'>Tap to toggle</span>"))
	C.w_align(sub, 0, 0.5)
	C.w_pack(vbd, sub, 0, 0, 0)

	d.brightnessScale = C.w_hscale(0, 2399, 1)
	C.w_size(d.brightnessScale, 120, -1)
	C.w_pack(vbd, d.brightnessScale, 1, 1, 0)
	d.brightnessVal = C.w_lbl()
	C.w_pack(vbd, d.brightnessVal, 0, 0, 0)

	ab := C.w_btn(C.CString("Apply"))
	C.w_signal(ab, C.CString("clicked"), C.GCallback(unsafe.Pointer(C.onApplyBrightness)))
	C.w_pack(vbd, ab, 0, 0, 0)

	br := C.w_hbox(0, 4)
	bl := C.w_lbl()
	C.w_markup(bl, C.CString("<span font_desc='12'>Battery</span>"))
	C.w_pack(br, bl, 0, 0, 0)
	d.batteryVal = C.w_lbl()
	C.w_pack_end(br, d.batteryVal, 1, 1, 0)
	C.w_pack(vbd, br, 0, 0, 0)
	C.w_add(dc, vbd)
	C.w_pack(btm, dc, 1, 1, 0)

	// Middle col: Mail + Agenda
	mc := C.w_vbox(0, 6)
	C.w_pack(mc, placeholderCard("Mail", "0", "No unread mail"), 1, 1, 0)
	C.w_pack(mc, placeholderCard("Agenda", "0", "No upcoming events"), 1, 1, 0)
	C.w_pack(btm, mc, 1, 1, 0)

	// Right col: Music + Connection
	rc := C.w_vbox(0, 6)
	C.w_pack(rc, placeholderCard("Music", "--", "Waiting for PC..."), 1, 1, 0)
	C.w_pack(rc, placeholderCard("Connection", "", "Home Assistant: Disconnected"), 0, 0, 0)
	C.w_pack(btm, rc, 1, 1, 0)

	C.w_pack(vb, btm, 1, 1, 0)
	return vb
}

func (d *Dashboard) showView(idx int) {
	for i, v := range d.views {
		if i == idx {
			C.w_show(v)
			C.w_markup(d.indicators[i], C.CString("<span font_desc='14'>●</span>"))
			C.w_fg(d.indicators[i], C.CString("#252525"))
		} else {
			C.w_hide(v)
			C.w_markup(d.indicators[i], C.CString("<span font_desc='14'>●</span>"))
			C.w_fg(d.indicators[i], C.CString("#d9d1bf"))
		}
	}
	d.currentView = idx
}

func (d *Dashboard) UpdateClock(now time.Time) {
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

	C.w_markup(d.greeting, C.CString(fmt.Sprintf("<span font_desc='10' weight='bold' color='#626262'>%s</span>", greet)))
	C.w_markup(d.clockLbl, C.CString(fmt.Sprintf("<span font_desc='72' weight='950'>%s</span>", now.Format("15:04"))))
	C.w_markup(d.dateLbl, C.CString(fmt.Sprintf("<span font_desc='18' weight='850'>%s</span>", now.Format("Monday, January 2"))))
	C.w_markup(d.calMonth, C.CString(fmt.Sprintf("<span font_desc='15' weight='950'>%s</span>", months[mm])))
	C.w_markup(d.calYear, C.CString(fmt.Sprintf("<span font_desc='11' weight='bold' color='#626262'>%d</span>", ym)))

	day := 1
	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			cell := d.calDays[r][c]
			if (r == 0 && c < sdow) || day > dim {
				C.w_markup(cell, C.CString("<span font_desc='9'> </span>"))
			} else {
				m := fmt.Sprintf("<span font_desc='9'>%d</span>", day)
				if day == dm {
					m = fmt.Sprintf("<span font_desc='9' weight='bold' foreground='#ffffff' background='#252525'> %d </span>", day)
				}
				C.w_markup(cell, C.CString(m))
				day++
			}
		}
	}

	if now.Second()%30 == 0 {
		batt := readBatteryCapacity()
		C.w_markup(d.batteryVal, C.CString(fmt.Sprintf("<span font_desc='12'>%s%%</span>", batt)))
		C.w_markup(d.greeting, C.CString(fmt.Sprintf("<span font_desc='10' weight='bold' color='#626262'>%s • Bat %s%%</span>", greet, batt)))
	}
}

// ── View helpers ──
func buildCalendarView() *C.GtkWidget {
	vb := C.w_vbox(0, 6)
	C.w_border(vb, 10)
	card := frameCard("Upcoming Agenda")
	sub := C.w_lbl()
	C.w_markup(sub, C.CString("<span font_desc='11' color='#626262'>Next 7 Days</span>"))
	C.w_add(card, sub)
	C.w_pack(vb, card, 0, 0, 0)
	emp := C.w_lbl()
	C.w_markup(emp, C.CString("<span font_desc='13'>Loading calendar...</span>"))
	C.w_pack(vb, emp, 1, 1, 0)
	return vb
}

func buildLauncherView() *C.GtkWidget {
	vb := C.w_vbox(0, 6)
	C.w_border(vb, 10)
	card := frameCard("PC Launcher")
	sub := C.w_lbl()
	C.w_markup(sub, C.CString("<span font_desc='10' color='#626262'>Remote Macro Controls</span>"))
	C.w_add(card, sub)
	C.w_pack(vb, card, 0, 0, 0)
	return vb
}

func frameCard(title string) *C.GtkWidget {
	f := C.w_frame(C.CString(title))
	C.w_shadow(f, C.GTK_SHADOW_ETCHED_IN)
	return f
}

func placeholderCard(title, badge, subtext string) *C.GtkWidget {
	f := frameCard(title)
	vb := C.w_vbox(0, 4)
	C.w_border(vb, 6)
	l := C.w_lbl()
	t := subtext
	if badge != "" && badge != "0" {
		t = fmt.Sprintf("[%s] %s", badge, subtext)
	}
	C.w_markup(l, C.CString(fmt.Sprintf("<span font_desc='10' color='#626262'>%s</span>", t)))
	C.w_pack(vb, l, 0, 0, 0)
	C.w_add(f, vb)
	return f
}
