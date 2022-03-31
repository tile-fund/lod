package proxy

import (
	"net/url"
	"strings"

	"github.com/dechristopher/lod/tile"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"

	"github.com/dechristopher/lod/cache"
	"github.com/dechristopher/lod/config"
	"github.com/dechristopher/lod/helpers"
	"github.com/dechristopher/lod/str"
	"github.com/dechristopher/lod/util"
)

type tileError struct {
	url   string
	proxy config.Proxy
}

// genHandler builds a new proxy endpoint handler from configuration
func genHandler(p config.Proxy) fiber.Handler {
	// preconfigure cache on boot
	c := cache.Get(p.Name)

	// handler function to wire to endpoint
	return func(ctx *fiber.Ctx) error {
		return handle(p, c, ctx)
	}
}

// handle proxy requests for the specified proxy config
func handle(p config.Proxy, cache *cache.Cache, ctx *fiber.Ctx) error {
	// check presence of configured URL parameters and store
	// their values in a map within the request locals
	helpers.FillParamsMap(p, ctx)

	// calculate url from the configured URL and params
	tileUrl, err := buildTileUrl(p, ctx)
	if err != nil {
		ctx.Locals("lod-cache", " :err ")
		util.Error(str.CProxy, str.EBadRequest, err.Error())
		return ctx.Status(fiber.StatusBadRequest).SendString("")
	}

	// calculate the cache key for this request using XYZ and URL params
	cacheKey, err := helpers.BuildCacheKey(p, ctx)
	if err != nil {
		ctx.Locals("lod-cache", "  :err")
		util.Error(str.CProxy, str.ECacheBuildKey, err.Error())
		return ctx.Status(fiber.StatusInternalServerError).SendString("")
	}

	if cachedTile := cache.Fetch(cacheKey, ctx); cachedTile != nil {
		// IF WE HIT A CACHED TILE
		// write the tile to the response body
		_, err := ctx.Write(cachedTile.TileData())
		if err != nil {
			ctx.Locals("lod-cache", "  :err")
			util.Error(str.CProxy, str.EWrite, err.Error(), tileError{
				url:   tileUrl,
				proxy: p,
			})
			return ctx.Status(fiber.StatusInternalServerError).SendString("")
		}

		// set stored headers in response
		for key, val := range cachedTile.Headers() {
			ctx.Set(key, val)
		}
	} else {
		// IF WE MISSED A CACHED TILE
		ctx.Locals("lod-cache", " :miss ")

		// inject headers to upstream request if any are configured
		if len(p.AddHeaders) > 0 {
			for _, header := range p.AddHeaders {
				ctx.Request().Header.Del(header.Name)
				ctx.Request().Header.Add(header.Name, header.Value)
			}
		}

		// perform request to tile URL
		if err := proxy.Do(ctx, tileUrl); err != nil {
			return err
		}

		if ctx.Response().StatusCode() == fiber.StatusNoContent ||
			(len(ctx.Response().Body()) > 0 && ctx.Response().StatusCode() == fiber.StatusOK) {
			// copy tile data into separate slice, so we don't lose the reference
			tileData := make([]byte, len(ctx.Response().Body()))
			copy(tileData, ctx.Response().Body())

			headers := map[string]string{}
			// Store configured headers into the tile cache for this tile
			p.PopulateHeaders(ctx, headers)

			// Delete headers from the final response that are on the DelHeaders list
			// if we got them from the tileserver. This can be used to prevent leaking
			// internals of the tileserver if you don't control what it returns
			p.DeleteHeaders(ctx)

			// set 204 Status No Content if upstream tileserver returned no/empty tile
			if ctx.Response().StatusCode() == fiber.StatusNoContent {
				ctx.Status(fiber.StatusNoContent)
			}

			// spin off a routine to cache the tile without blocking the response
			go cache.EncodeSet(cacheKey, tileData, headers)
		} else {
			ctx.Locals("lod-cache", " :err-u")
			// Send internal server error response with empty body if upstream
			// fails to respond or responds with a non-200 status code
			return ctx.Status(fiber.StatusInternalServerError).SendString("")
		}
	}

	// Remove server header from response
	ctx.Response().Header.Del(fiber.HeaderServer)

	return ctx.Next()
}

// buildTileUrl will substitute URL tile params into the proxy tile URL
func buildTileUrl(proxy config.Proxy, ctx *fiber.Ctx) (string, error) {
	currentTile, err := helpers.GetTile(ctx)
	if err != nil {
		return "", err
	}

	// replace XYZ values in the tile URL
	baseUrl := currentTile.InjectString(proxy.TileURL)

	// replace dynamic endpoint parameter in URL if configured
	if proxy.HasEParam {
		endpoint := ctx.Params("e")
		baseUrl = strings.ReplaceAll(baseUrl, tile.EndpointTemplate, endpoint)
	}

	// fetch params from context for possible addition to URL
	paramsMap := helpers.GetParamsFromCtx(ctx)

	// if no query parameters, return baseUrl
	if paramsMap == nil {
		return baseUrl, nil
	}

	// parse baseURL to add URL parameters
	paramUrl, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	// replace params by name in the key template if any exist
	for param, val := range paramsMap {
		params.Add(param, val)
	}

	// set encoded params in URL
	paramUrl.RawQuery = params.Encode()

	// return generated URL with substitutions for query parameters
	return paramUrl.String(), nil
}
