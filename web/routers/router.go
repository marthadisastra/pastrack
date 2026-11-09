package routers

import (
	"github.com/beego/beego/v2/server/web"
	"io/fs"
	"net/http"
	"pastrack/web/controllers"
	"pastrack/web/ui"
)

func Register() {
	web.BConfig.RunMode = "prod"
	web.BConfig.WebConfig.AutoRender = false
	web.BConfig.WebConfig.EnableDocs = false
	web.BConfig.WebConfig.DirectoryIndex = false
	web.Router("/", &controllers.Dashboard{})
	web.Router("/login", &controllers.Login{})
	web.Router("/logout", &controllers.Logout{})
	web.Router("/api/users", &controllers.Users{})
	web.Router("/api/users/:id", &controllers.Users{})
	web.Router("/api/users/:id/restore", &controllers.Users{}, "post:Restore")
	web.Router("/api/users/:id/password", &controllers.Users{}, "post:ResetPassword")
	web.Router("/api/me", &controllers.Me{})
	web.Router("/api/me/password", &controllers.Me{}, "post:Password")
	web.Router("/api/master/sensors", &controllers.FuelSensorMaster{})
	web.Router("/api/master/sensors/:id", &controllers.FuelSensorMaster{})
	web.Router("/api/master/sensors/:id/restore", &controllers.FuelSensorMaster{}, "post:Restore")
	web.Router("/api/master/:entity", &controllers.Master{})
	web.Router("/api/master/:entity/:id", &controllers.Master{})
	web.Router("/api/master/:entity/:id/restore", &controllers.Master{}, "post:Restore")
	web.Router("/api/vehicles/:id/photo", &controllers.VehiclePhoto{})
	web.Router("/api/fuel-calibration/:id", &controllers.FuelCalibration{})
	web.Router("/api/fuel-calibration/:id/capture/:level", &controllers.FuelCapture{}, "post:Post")
	web.Router("/api/fuel-calibration/:id/points/:point", &controllers.FuelCalibration{}, "delete:Delete")
	web.Router("/api/dashboards/:kind", &controllers.AnalyticsDashboard{})
	web.Router("/api/reports", &controllers.Report{})
	web.Router("/api/tracking/address", &controllers.TrackingAddress{})
	web.Router("/api/tracking/places", &controllers.TrackingPlaces{})
	web.Router("/api/tracking", &controllers.Tracking{})
	web.Router("/api/notifications", &controllers.NotificationAPI{})
	web.Handler("/api/realtime", http.HandlerFunc(controllers.RealtimeStream))
	web.Router("/firebase-messaging-sw.js", &controllers.FirebaseWorker{})

	web.Router("/api/audit", &controllers.AuditView{})
	web.Router("/api/:kind", &controllers.API{})
	web.Router("/healthz", &controllers.Health{})
	sub, _ := fs.Sub(ui.Assets, "static")
	web.Handler("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(sub))), true)

}
