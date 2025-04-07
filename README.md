# pomodoro-inkywhat
A pomodoro timer built for inky what

## Aims:
- Use an inky what display (eink 400x300 resolution)
- Use a raspberry pi (zero2 w)
- Use a single button to start/reset a timer
- When not in timer mode, display the time and weather conditions

## General approach

Not sure if this will work, but I'm going to try and keep this as simple as:

1. Create individual image panels using imagemagick
2. Mash the images together using imagemagick
3. Display the image on the inky what using the periph.io device libraries
4. Single button to start a 15 minute timer, same button to reset the timer

https://simplemaps.com/data/us-zips
