package standard

import (
	"sort"

	"github.com/MagicalTux/goro/core"
	"github.com/MagicalTux/goro/core/phpctx"
	"github.com/MagicalTux/goro/core/phpv"
)

// > func void header ( string $header [, bool $replace = TRUE [, int $response_code ]] )
func fncHeader(ctx phpv.Context, args []*phpv.ZVal) (*phpv.ZVal, error) {
	var line string
	var replace *bool
	var responseCode *phpv.ZInt
	_, err := core.Expand(ctx, args, &line, &replace, &responseCode)
	if err != nil {
		return nil, err
	}

	replaceValue := true
	if replace != nil {
		replaceValue = *replace
	}

	code := 0
	if responseCode != nil {
		code = int(*responseCode)
	}

	err = ctx.Global().(*phpctx.Global).AddResponseHeaderLine(line, replaceValue, code)
	if err != nil {
		return nil, err
	}
	return phpv.ZNULL.ZVal(), nil
}

// > func void header_remove ([ string $name ] )
func fncHeaderRemove(ctx phpv.Context, args []*phpv.ZVal) (*phpv.ZVal, error) {
	var name *string
	_, err := core.Expand(ctx, args, &name)
	if err != nil && err != core.ErrNotEnoughArguments {
		return nil, err
	}

	value := ""
	if name != nil {
		value = *name
	}
	ctx.Global().(*phpctx.Global).RemoveResponseHeader(value)
	return phpv.ZNULL.ZVal(), nil
}

// > func array headers_list ( void )
func fncHeadersList(ctx phpv.Context, args []*phpv.ZVal) (*phpv.ZVal, error) {
	headers := ctx.Global().(*phpctx.Global).ResponseHeaders()
	result := phpv.NewZArray()
	if len(headers) == 0 {
		return result.ZVal(), nil
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for index := 0; index < len(keys); index++ {
		values := headers[keys[index]]
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			line := keys[index] + ": " + values[valueIndex]
			if err := result.OffsetSet(ctx, nil, phpv.ZString(line).ZVal()); err != nil {
				return nil, err
			}
		}
	}
	return result.ZVal(), nil
}

// > func bool headers_sent ( void )
func fncHeadersSent(ctx phpv.Context, args []*phpv.ZVal) (*phpv.ZVal, error) {
	return phpv.ZBool(ctx.Global().(*phpctx.Global).HeadersSent()).ZVal(), nil
}

// > func int|bool http_response_code ([ int $response_code ] )
func fncHTTPResponseCode(ctx phpv.Context, args []*phpv.ZVal) (*phpv.ZVal, error) {
	global := ctx.Global().(*phpctx.Global)
	current := global.ResponseStatusCode()
	if len(args) == 0 {
		if current == 0 {
			return phpv.ZBool(false).ZVal(), nil
		}
		return phpv.ZInt(current).ZVal(), nil
	}

	var responseCode phpv.ZInt
	_, err := core.Expand(ctx, args, &responseCode)
	if err != nil {
		return nil, err
	}
	global.SetResponseStatus(int(responseCode))

	if current == 0 {
		return phpv.ZBool(true).ZVal(), nil
	}
	return phpv.ZInt(current).ZVal(), nil
}
