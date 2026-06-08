package main

/*
#cgo LDFLAGS: -lgtk-x11-2.0 -lgdk-x11-2.0 -lgdk_pixbuf-2.0 -lpangocairo-1.0 -lpango-1.0 -lcairo -latk-1.0 -lgio-2.0 -lgobject-2.0 -lglib-2.0 -lX11 -lXext -lXrender
#cgo CFLAGS: -I/usr/include/gtk-2.0 -I/usr/include/gdk-2.0 -I/usr/include/atk-1.0 -I/usr/include/cairo -I/usr/include/pango-1.0 -I/usr/include/gdk-pixbuf-2.0 -I/usr/include/glib-2.0 -I/usr/include/pixman-1 -I/usr/include/freetype2 -I/usr/include/libpng16 -I/usr/lib/glib-2.0/include -I/usr/lib/gtk-2.0/include

#include <gtk/gtk.h>
#include <gdk/gdk.h>
#include <X11/Xlib.h>
#include <string.h>
#include <stdlib.h>

extern void goQuitClicked();
extern void goApplyBrightness();
extern void goDestroy();

// Widget creation helpers
static GtkWidget* make_window(void) {
    GtkWidget *win = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    gtk_window_set_title(GTK_WINDOW(win), "Kindle Dashboard");
    gtk_widget_set_size_request(win, 600, 800);
    gtk_window_set_resizable(GTK_WINDOW(win), FALSE);
    gtk_window_set_decorated(GTK_WINDOW(win), FALSE);
    gtk_window_set_keep_above(GTK_WINDOW(win), TRUE);
    gtk_window_move(GTK_WINDOW(win), 0, 0);
    gtk_container_set_border_width(GTK_CONTAINER(win), 0);
    return win;
}

// Called periodically in idle to find and configure our window
// Uses raw Xlib since GDK X11 macros not available in GTK 2.10
static gboolean configure_window(gpointer data) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) return TRUE; /* keep trying */
    Window root = DefaultRootWindow(dpy);
    Window parent;
    Window *children;
    unsigned int nchildren;
    gboolean found = FALSE;
    if (XQueryTree(dpy, root, &root, &parent, &children, &nchildren)) {
        for (unsigned int i = 0; i < nchildren; i++) {
            char *name = NULL;
            if (XFetchName(dpy, children[i], &name) && name) {
                if (strstr(name, "Kindle Dashboard") != NULL) {
                    XSetWindowAttributes attrs;
                    attrs.override_redirect = True;
                    XChangeWindowAttributes(dpy, children[i], CWOverrideRedirect, &attrs);
                    XMapRaised(dpy, children[i]);
                    XFree(name);
                    found = TRUE;
                    break;
                }
                XFree(name);
            }
        }
        if (children) XFree(children);
    }
    XFlush(dpy);
    XCloseDisplay(dpy);
    return !found; /* keep trying until found */
}

// Start the idle callback
static void start_configure_idle(void) {
    g_idle_add(configure_window, NULL);
}

static GtkWidget* make_vbox(void) {
    return gtk_vbox_new(FALSE, 8);
}

static GtkWidget* make_hscale(double min, double max, double step) {
    GtkObject *adj = gtk_adjustment_new(min, min, max, step, step*10, 0);
    return gtk_hscale_new(GTK_ADJUSTMENT(adj));
}

static double get_scale_value(GtkWidget *scale) {
    return gtk_range_get_value(GTK_RANGE(scale));
}

static void box_pack(GtkWidget *box, GtkWidget *child, gboolean expand, gboolean fill, guint padding) {
    gtk_box_pack_start(GTK_BOX(box), child, expand, fill, padding);
}

static void connect_signal(GtkWidget *widget, const char *signal, GCallback cb) {
    g_signal_connect(G_OBJECT(widget), signal, cb, NULL);
}
*/
import "C"
import (
	"fmt"
	"os"
	"time"
	"unsafe"
)

var (
	clockLabel     *C.GtkWidget
	dateLabel      *C.GtkWidget
	batteryLabel   *C.GtkWidget
	brightnessScale *C.GtkWidget
	labelBrightnessVal *C.GtkWidget
)

//export goQuitClicked
func goQuitClicked() {
	C.gtk_main_quit()
}

//export goDestroy
func goDestroy() {
	C.gtk_main_quit()
}

//export goApplyBrightness
func goApplyBrightness() {
	val := int(C.get_scale_value(brightnessScale))
	writeBrightness(val)

	// Update the label
	cs := C.CString(fmt.Sprintf("Set to %d", val))
	C.gtk_label_set_text((*C.GtkLabel)(unsafe.Pointer(labelBrightnessVal)), cs)
	C.free(unsafe.Pointer(cs))
}

func main() {
	C.gtk_init(nil, nil)

	window := C.make_window()
	C.connect_signal(window, C.CString("destroy"), C.GCallback(unsafe.Pointer(C.goDestroy)))

	vbox := C.make_vbox()

	// Clock
	clockLabel = C.gtk_label_new(nil)
	C.gtk_label_set_markup((*C.GtkLabel)(unsafe.Pointer(clockLabel)),
		C.CString("<span font_desc='64' weight='950'>--:--</span>"))
	C.box_pack(vbox, clockLabel, 0, 0, 0)

	// Date
	dateLabel = C.gtk_label_new(nil)
	C.gtk_label_set_markup((*C.GtkLabel)(unsafe.Pointer(dateLabel)),
		C.CString("<span font_desc='20' weight='850'>Loading...</span>"))
	C.box_pack(vbox, dateLabel, 0, 0, 0)

	// Calendar
	cal := C.gtk_calendar_new()
	C.box_pack(vbox, cal, 0, 0, 0)

	// Brightness frame
	brightFrame := C.gtk_frame_new(C.CString("Brightness"))
	brightVBox := C.make_vbox()

	maxB := readMaxBrightness()
	brightnessScale = C.make_hscale(0.0, C.double(maxB), 1.0)
	C.box_pack(brightVBox, brightnessScale, 0, 0, 0)

	labelBrightnessVal = C.gtk_label_new(C.CString(fmt.Sprintf("Max: %d", maxB)))
	C.box_pack(brightVBox, labelBrightnessVal, 0, 0, 0)

	applyBtn := C.gtk_button_new_with_label(C.CString("Apply"))
	C.connect_signal(applyBtn, C.CString("clicked"), C.GCallback(unsafe.Pointer(C.goApplyBrightness)))
	C.box_pack(brightVBox, applyBtn, 0, 0, 0)

	C.gtk_container_add((*C.GtkContainer)(unsafe.Pointer(brightFrame)), brightVBox)
	C.box_pack(vbox, brightFrame, 0, 0, 0)

	// Battery
	batteryLabel = C.gtk_label_new(C.CString("Battery: --"))
	C.box_pack(vbox, batteryLabel, 0, 0, 0)

	// Quit button
	quitBtn := C.gtk_button_new_with_label(C.CString("Quit"))
	C.connect_signal(quitBtn, C.CString("clicked"), C.GCallback(unsafe.Pointer(C.goQuitClicked)))
	C.box_pack(vbox, quitBtn, 0, 0, 0)

	C.gtk_container_add((*C.GtkContainer)(unsafe.Pointer(window)), vbox)
	C.gtk_widget_show_all(window)
	C.start_configure_idle()

	// Periodic update from a goroutine
	go func() {
		for {
			now := time.Now()
			timeStr := now.Format("15:04:05")
			dateStr := now.Format("Mon, Jan 2")

			cs1 := C.CString(fmt.Sprintf("<span font_desc='64' weight='950'>%s</span>", timeStr))
			C.gtk_label_set_markup((*C.GtkLabel)(unsafe.Pointer(clockLabel)), cs1)
			C.free(unsafe.Pointer(cs1))

			cs2 := C.CString(fmt.Sprintf("<span font_desc='20' weight='850'>%s</span>", dateStr))
			C.gtk_label_set_markup((*C.GtkLabel)(unsafe.Pointer(dateLabel)), cs2)
			C.free(unsafe.Pointer(cs2))

			if now.Second()%30 == 0 {
				batt := readBattery("capacity")
				cs3 := C.CString(fmt.Sprintf("Battery: %s%%", batt))
				C.gtk_label_set_text((*C.GtkLabel)(unsafe.Pointer(batteryLabel)), cs3)
				C.free(unsafe.Pointer(cs3))
			}

			time.Sleep(1 * time.Second)
		}
	}()

	C.gtk_main()
}

// Sysfs helpers
func readMaxBrightness() int {
	return readIntFile("/sys/devices/soc0/bl/backlight/bl/max_brightness", 2399)
}

func writeBrightness(val int) {
	data := []byte(fmt.Sprintf("%d", val))
	os.WriteFile("/sys/devices/soc0/bl/backlight/bl/brightness", data, 0644)
}

func readBattery(field string) string {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/power_supply/bd71827_bat/%s", field))
	if err != nil {
		return "N/A"
	}
	return string(data)
}

func readIntFile(path string, def int) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	var val int
	fmt.Sscanf(string(data), "%d", &val)
	return val
}
