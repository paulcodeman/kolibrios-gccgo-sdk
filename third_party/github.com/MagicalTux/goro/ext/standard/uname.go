package standard

import (
	"runtime"

	"github.com/MagicalTux/goro/core"
	"github.com/MagicalTux/goro/core/phpv"
)

// fallback uname

func fncUname(ctx phpv.Context, args []*phpv.ZVal) (*phpv.ZVal, error) {
	const hostname = "kolibrios"

	var arg string
	_, err := core.Expand(ctx, args, &arg)
	if err != nil {
		return nil, err
	}

	switch arg {
	case "s":
		return phpv.ZString(runtime.GOOS).ZVal(), nil
	case "n":
		return phpv.ZString(hostname).ZVal(), nil
	case "r":
		return phpv.ZString("?").ZVal(), nil
	case "m":
		return phpv.ZString(runtime.GOARCH).ZVal(), nil
	default:
		fallthrough
	case "a":
		// return full uname, ie "s n r v m"
		return phpv.ZString(runtime.GOOS + " " + hostname + " " + runtime.GOARCH).ZVal(), nil
	}
}
