import SwiftUI

struct DueNotebook: View {
    let model: DueModel
    let logContact: (DueJob) -> Void
    let openJob: (DueJob) -> Void
    let openApplication: (DueJob) -> Void
    @State private var opened: Set<String>?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                header
                if !model.comingUp.isEmpty { upcoming }
                VStack(spacing: 10) {
                    ForEach(model.dueJobs) { job in
                        DueJobCard(job: job, model: model, isOpen: opened?.contains(job.id) ?? false,
                                   toggle: { toggle(job) }, logContact: logContact, openJob: openJob,
                                   openApplication: openApplication)
                            .transition(.opacity)
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 8)
            .padding(.bottom, 24)
            .animation(.snappy(duration: 0.25), value: model.jobs)
        }
        .background(Color(.systemBackground))
        .onAppear {
            if opened == nil { opened = Set(model.dueJobs.prefix(1).map(\.id)) }
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(Date.now.formatted(.dateTime.weekday(.wide).day().month(.wide).locale(Locale(identifier: "en_GB"))))
                .font(.system(.title, design: .serif))
                .accessibilityAddTraits(.isHeader)
            Text(LocalizedStringKey(summary))
                .font(.footnote)
                .foregroundStyle(.secondary)
                .contentTransition(.numericText())
        }
    }

    private var summary: String {
        let tasks = model.dueJobs.reduce(0) { $0 + $1.tasks.count }
        let parts = [
            model.chasesDue > 0 ? "^[\(model.chasesDue) chase](inflect: true)" : nil,
            model.readyDrafts > 0 ? "\(model.readyDrafts) CV ready" : nil,
            tasks > 0 ? "^[\(tasks) task](inflect: true)" : nil,
        ].compactMap { $0 }
        return parts.isEmpty ? "Nothing due" : parts.joined(separator: ", ")
    }

    private var upcoming: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text("Coming up")
                Spacer()
                Text(model.comingUp.count, format: .number).monospacedDigit()
            }
            .font(.footnote.weight(.medium))
            .foregroundStyle(.secondary)
            ScrollView(.horizontal) {
                HStack(spacing: 6) {
                    ForEach(model.comingUp) { job in
                        HStack(spacing: 6) {
                            Text(job.chaseBy?.formatted(.dateTime.day().month(.abbreviated).locale(Locale(identifier: "en_GB"))) ?? "")
                                .font(.caption.monospacedDigit())
                                .foregroundStyle(.orange)
                            Text("Chase \(job.companyName)").font(.footnote.weight(.medium))
                        }
                        .padding(.horizontal, 10)
                        .frame(minHeight: 44)
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(Color(.separator)))
                        .accessibilityElement(children: .combine)
                    }
                }
            }
            .scrollIndicators(.hidden)
        }
    }

    private func toggle(_ job: DueJob) {
        var set = opened ?? []
        if set.contains(job.id) { set.remove(job.id) } else { set.insert(job.id) }
        withAnimation(.snappy(duration: 0.22)) { opened = set }
    }
}
