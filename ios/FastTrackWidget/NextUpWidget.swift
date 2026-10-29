import SwiftUI
import WidgetKit

struct NextUpEntry: TimelineEntry {
    let date: Date
    let next: NextUp
    let writtenAt: Date?
    let isStale: Bool
}

struct NextUpProvider: TimelineProvider {
    typealias Entry = NextUpEntry

    func placeholder(in context: Context) -> NextUpEntry {
        NextUpEntry(date: .now, next: .allClear, writtenAt: nil, isStale: false)
    }

    func getSnapshot(in context: Context, completion: @escaping (NextUpEntry) -> Void) {
        completion(entries(at: .now, calendar: .current)[0])
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<NextUpEntry>) -> Void) {
        let entries = entries(at: .now, calendar: .current)
        completion(Timeline(entries: entries, policy: .after(entries.last?.date ?? .now)))
    }

    private func entries(at now: Date, calendar: Calendar) -> [NextUpEntry] {
        let snapshot = NextUpSnapshot.sharedFileURL.flatMap(NextUpSnapshot.read(from:))
        return NextUpSnapshot.refreshDates(from: now, calendar: calendar).map { date in
            guard let snapshot else { return NextUpEntry(date: date, next: .allClear, writtenAt: nil, isStale: false) }
            return NextUpEntry(date: date, next: NextUp(snapshot, at: date, calendar: calendar),
                               writtenAt: snapshot.writtenAt, isStale: snapshot.isStale(at: date))
        }
    }
}

struct NextUpView: View {
    let entry: NextUpEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            switch entry.next {
            case .chase(let item, let days):
                Text("Chase \(item.companyName) · \(days == 0 ? "due today" : "\(days)d over")").font(.headline)
            case .apply(let item):
                Text(item.title).font(.headline).lineLimit(3)
                Text(item.companyName).font(.subheadline).foregroundStyle(.secondary)
            case .comingUp(let item):
                Text("Nothing due").font(.headline)
                Text("next: chase \(item.companyName)\(item.chaseBy.map { " " + $0.formatted(.dateTime.weekday(.abbreviated)) } ?? "")")
                    .font(.subheadline).foregroundStyle(.secondary)
            case .allClear:
                Text("All clear").font(.headline)
            }
            Spacer(minLength: 0)
            if entry.isStale, let writtenAt = entry.writtenAt {
                Text("as of \(writtenAt.formatted(.dateTime.weekday(.abbreviated)))")
                    .font(.caption2).foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .widgetURL(entry.next.focus?.url ?? DueFocus.dueTabURL)
        .containerBackground(.fill.tertiary, for: .widget)
    }
}

struct NextUpWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: NextUpSnapshot.widgetKind, provider: NextUpProvider()) { entry in
            NextUpView(entry: entry)
        }
        .configurationDisplayName("Next up")
        .description("The one thing to do now.")
        .supportedFamilies([.systemSmall])
    }
}
