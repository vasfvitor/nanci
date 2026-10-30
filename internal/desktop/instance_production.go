//go:build production

package main

// singleInstanceID names the lock that keeps one Nanci open per user session.
// The production build (wails3 build, -tags production) and the dev build use
// different IDs, so wails3 dev runs while the installed app is open.
const singleInstanceID = "com.wails.nanci-desktop"
