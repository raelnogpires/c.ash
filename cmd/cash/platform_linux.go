//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0 gdk-pixbuf-2.0
#include <gtk/gtk.h>
#include <stdlib.h>

typedef struct {
	guchar *data;
	gsize length;
} CashIconUpdate;

static gboolean cash_apply_window_icon(gpointer user_data) {
	CashIconUpdate *update = (CashIconUpdate *)user_data;
	GdkPixbufLoader *loader = gdk_pixbuf_loader_new();
	GError *error = NULL;

	if (gdk_pixbuf_loader_write(loader, update->data, update->length, &error) &&
		gdk_pixbuf_loader_close(loader, &error)) {
		GdkPixbuf *pixbuf = gdk_pixbuf_loader_get_pixbuf(loader);
		if (pixbuf != NULL) {
			GList *windows = gtk_window_list_toplevels();
			for (GList *item = windows; item != NULL; item = item->next) {
				if (GTK_IS_WINDOW(item->data)) {
					gtk_window_set_icon(GTK_WINDOW(item->data), pixbuf);
				}
			}
			g_list_free(windows);
		}
	}

	if (error != NULL) g_error_free(error);
	g_object_unref(loader);
	free(update->data);
	free(update);
	return G_SOURCE_REMOVE;
}

static void cash_schedule_window_icon(guchar *data, gsize length) {
	CashIconUpdate *update = malloc(sizeof(CashIconUpdate));
	update->data = data;
	update->length = length;
	g_idle_add(cash_apply_window_icon, update);
}
*/
import "C"

import (
	"os"

	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

func configureWebView() {
	// Recent WebKitGTK releases may open a blank, non-interactive surface when
	// DMA-BUF rendering is used with older Intel/i915 graphics. Respect an
	// explicit user choice, but prefer the reliable renderer by default.
	if _, configured := os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER"); !configured {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
}

func platformOptions(icon []byte) *linux.Options {
	return &linux.Options{Icon: icon, ProgramName: "cash"}
}

func setPlatformIcon(icon []byte) {
	if len(icon) == 0 {
		return
	}
	data := C.CBytes(icon)
	C.cash_schedule_window_icon((*C.guchar)(data), C.gsize(len(icon)))
	// Ownership moves to cash_apply_window_icon, which runs on GTK's main loop.
}
