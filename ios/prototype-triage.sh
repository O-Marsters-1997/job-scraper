#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
DEVICE=${DEVICE:-booted}
DERIVED=build/prototype
xcodebuild -project FastTrack.xcodeproj -scheme FastTrack -configuration Debug \
	-destination "generic/platform=iOS Simulator" -derivedDataPath "$DERIVED" -quiet build
xcrun simctl install "$DEVICE" "$DERIVED/Build/Products/Debug-iphonesimulator/FastTrack.app"
xcrun simctl launch --terminate-running-process "$DEVICE" com.ollymarsters.fasttrack -prototype-triage
open -a Simulator
