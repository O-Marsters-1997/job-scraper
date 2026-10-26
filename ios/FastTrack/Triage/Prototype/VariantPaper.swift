#if DEBUG
import SwiftUI

struct VariantPaper: View {
    let store: TriagePrototypeStore

    private var shortlisted: Int { store.log.filter { $0.removed && $0.action.isPositive }.count }

    var body: some View {
        VStack(spacing: 0) {
            TriageCardHost(store: store) { job, isExpanded, toggle in
                page(job, isExpanded: isExpanded, toggle: toggle)
            }
            if let job = store.current {
                bar(job)
            }
        }
        .background(Color(.systemBackground))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                VStack(spacing: 0) {
                    Text("Triage").font(.headline)
                    Text("\(shortlisted) shortlisted")
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(.secondary)
                        .contentTransition(.numericText(value: Double(shortlisted)))
                        .animation(.snappy, value: shortlisted)
                }
            }
        }
    }

    private func page(_ job: PrototypeJob, isExpanded: Bool, toggle: @escaping () -> Void) -> some View {
        Group {
            if isExpanded {
                ScrollView {
                    VStack(alignment: .leading, spacing: 32) {
                        header(job)
                        Button("Back to summary", systemImage: "chevron.up", action: toggle)
                            .font(.subheadline.weight(.medium))
                        JobDetailBody(job: job)
                    }
                    .padding(.horizontal, 24)
                    .padding(.vertical, 20)
                }
            } else {
                VStack(alignment: .leading, spacing: 28) {
                    header(job)
                    evidence(job)
                    Spacer(minLength: 0)
                    Button(action: toggle) {
                        Label("Read the listing", systemImage: "chevron.down")
                            .font(.subheadline.weight(.medium))
                    }
                }
                .padding(.horizontal, 24)
                .padding(.vertical, 20)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
        }
        .background(Color(.systemBackground))
        .accessibilityElement(children: isExpanded ? .contain : .combine)
    }

    private func header(_ job: PrototypeJob) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text(job.company)
                Spacer()
                Text(job.age).monospacedDigit()
            }
            .font(.subheadline.weight(.medium))
            .foregroundStyle(.secondary)
            Text(job.title)
                .font(.largeTitle.weight(.bold))
                .tracking(-0.4)
                .lineLimit(4)
                .minimumScaleFactor(0.85)
            Text("\(job.location) · \(job.arrangement)")
                .font(.body)
                .foregroundStyle(.secondary)
            if let salary = job.salary {
                Text(salary).font(.title3.monospacedDigit())
            }
        }
    }

    private func evidence(_ job: PrototypeJob) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 8) {
                Circle().fill(job.band.colour).frame(width: 8, height: 8)
                Text(job.band.rawValue).font(.subheadline.weight(.semibold))
                Text(job.score, format: .number)
                    .font(.subheadline.monospacedDigit())
                    .foregroundStyle(.secondary)
                if job.unknownCount > 0 {
                    Text("· \(job.unknownCount) unknown").font(.subheadline).foregroundStyle(.secondary)
                }
            }
            Rectangle().fill(Color(.separator)).frame(height: 1 / 3)
            ForEach(job.topRows, id: \.self) { BreakdownRowView(row: $0) }
        }
    }

    private func bar(_ job: PrototypeJob) -> some View {
        HStack {
            Button("Dismiss") { act(.dismiss, job) }
                .font(.body.weight(.medium))
                .foregroundStyle(.secondary)
                .frame(minWidth: 88, minHeight: 50)
            Spacer()
            TriageMoreMenu(job: job, store: store) {
                Image(systemName: "ellipsis")
                    .font(.body.weight(.semibold))
                    .frame(width: 50, height: 50)
                    .accessibilityLabel("More")
            }
            .foregroundStyle(.secondary)
            Spacer()
            Button { act(.shortlist, job) } label: {
                Text("Shortlist")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 28)
                    .frame(minHeight: 50)
                    .background(Color.brandTint, in: .capsule)
            }
        }
        .padding(.horizontal, 24)
        .padding(.vertical, 8)
    }

    private func act(_ action: TriagePrototypeStore.Action, _ job: PrototypeJob) {
        withAnimation(.snappy) { store.act(action, on: job) }
    }
}
#endif
