package ui

import (
	"fmt"
	"io"
	"strings"
)

const bannerArt = `
████████╗███████╗ ██████╗ ██╗   ██╗ █████╗ ██████╗ ██████╗ 
╚══██╔══╝██╔════╝██╔════╝ ██║   ██║██╔══██╗██╔══██╗██╔══██╗
   ██║   █████╗  ██║  ███╗██║   ██║███████║██████╔╝██║  ██║
   ██║   ██╔══╝  ██║   ██║██║   ██║██╔══██║██╔══██╗██║  ██║
   ██║   ██║     ╚██████╔╝╚██████╔╝██║  ██║██║  ██║██████╔╝
   ╚═╝   ╚═╝      ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ 
`

// Banner writes the tfguard logo and a short tagline.
func Banner(w io.Writer, style Style, version string) {
	logo := strings.TrimPrefix(bannerArt, "\n")
	if style.Enabled() {
		fmt.Fprint(w, style.Cyan(logo))
	} else {
		fmt.Fprint(w, logo)
	}
	tag := fmt.Sprintf("  terraform plan guardian  ·  v%s", version)
	fmt.Fprintln(w, style.Dim(tag))
	fmt.Fprintln(w)
}
