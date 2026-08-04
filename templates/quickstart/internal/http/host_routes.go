package http

import "github.com/gin-gonic/gin"

// registerHostRoutes is the entrypoint for host-defined business routes.
// Mount your own handler packages from here and keep shared stack routes in
// go-modules/stacks/saascore unchanged.
func registerHostRoutes(publicGroup, userGroup *gin.RouterGroup) {
	_ = publicGroup
	_ = userGroup
}
