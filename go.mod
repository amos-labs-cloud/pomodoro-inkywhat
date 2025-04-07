module github.com/amos-labs-cloud/pomodoro-inkywhat

go 1.23.4

replace periph.io/x/devices/v3 => github.com/gsexton/devices/v3 v3.6.12-0.20250406043622-e455f8e488dc

require (
	github.com/icodealot/noaa v0.0.2
	gopkg.in/gographics/imagick.v2 v2.7.0
	periph.io/x/devices/v3 v3.7.4
	periph.io/x/host/v3 v3.8.4
)

require periph.io/x/conn/v3 v3.7.2 // indirect
