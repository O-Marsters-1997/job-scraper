import SwiftUI
import WidgetKit

struct PlaceholderEntry: TimelineEntry {
	let date: Date
}

struct PlaceholderProvider: TimelineProvider {
	func placeholder(in context: Context) -> PlaceholderEntry {
		PlaceholderEntry(date: .now)
	}

	func getSnapshot(in context: Context, completion: @escaping (PlaceholderEntry) -> Void) {
		completion(PlaceholderEntry(date: .now))
	}

	func getTimeline(in context: Context, completion: @escaping (Timeline<PlaceholderEntry>) -> Void) {
		completion(Timeline(entries: [PlaceholderEntry(date: .now)], policy: .never))
	}
}

struct PlaceholderWidget: Widget {
	var body: some WidgetConfiguration {
		StaticConfiguration(kind: "PlaceholderWidget", provider: PlaceholderProvider()) { _ in
			Text("FastTrack")
				.containerBackground(.fill.tertiary, for: .widget)
		}
		.configurationDisplayName("FastTrack")
		.description("Placeholder.")
		.supportedFamilies([.systemSmall])
	}
}
