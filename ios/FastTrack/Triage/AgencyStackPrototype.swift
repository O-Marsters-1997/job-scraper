#if DEBUG
import SwiftUI

private enum ProtoKind { case direct, likelyDirect, unclassified, agency, likelyAgency
    var isAgency: Bool { self == .agency || self == .likelyAgency }
}

private struct ProtoJob: Identifiable, Hashable {
    let id = UUID()
    let company: String
    let title: String
    var kind: ProtoKind
}

private enum Stack: String { case direct = "Direct", agency = "Agency" }

@Observable
private final class ProtoModel {
    var jobs: [ProtoJob] = [
        .init(company: "Ostara Labs", title: "Senior Go Engineer", kind: .direct),
        .init(company: "Hays Technology", title: "Go Developer (Contract)", kind: .agency),
        .init(company: "Fenwick Digital", title: "Backend Engineer", kind: .likelyDirect),
        .init(company: "Harnham", title: "Platform Engineer", kind: .agency),
        .init(company: "Brightpath", title: "Software Engineer, Payments", kind: .likelyAgency),
        .init(company: "Brightpath", title: "Senior Software Engineer", kind: .likelyAgency),
        .init(company: "Kettle & Co", title: "Full-stack Engineer", kind: .unclassified),
        .init(company: "Opus Recruitment", title: "Lead Go Engineer", kind: .agency),
    ]
    var stack: Stack = .direct
    var banner: String?
    private var undoSnapshot: [ProtoJob]?

    func queue(_ s: Stack) -> [ProtoJob] { jobs.filter { $0.kind.isAgency == (s == .agency) } }
    var current: ProtoJob? { queue(stack).first }
    var otherCount: Int { queue(stack == .direct ? .agency : .direct).count }

    func decide(_ verb: String) {
        guard let job = current else { return }
        undoSnapshot = jobs
        jobs.removeAll { $0.id == job.id }
        banner = "\(verb) · \(job.title)"
    }

    func reclassify() {
        guard let job = current else { return }
        undoSnapshot = jobs
        let to: ProtoKind = job.kind.isAgency ? .direct : .agency
        let moved = jobs.indices.filter { jobs[$0].company == job.company }
        for i in moved { jobs[i].kind = to }
        banner = "Marked \(job.company) as \(to == .agency ? "an agency" : "direct") · \(moved.count) moved to \(to == .agency ? "Agency" : "Direct")"
    }

    func undo() {
        if let undoSnapshot { jobs = undoSnapshot }
        undoSnapshot = nil
        banner = nil
    }
}

private struct KindBadge: View {
    let kind: ProtoKind
    var body: some View {
        switch kind {
        case .agency: DotLabel(text: "Agency", hue: .orange)
        case .likelyAgency: DotLabel(text: "Maybe recruiter", hue: .yellow)
        default: EmptyView()
        }
    }
}

private struct ProtoCard: View {
    let job: ProtoJob
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                DotLabel(text: "Great", hue: .green)
                KindBadge(kind: job.kind)
                Spacer()
                Text("2d").font(.footnote).foregroundStyle(.secondary)
            }
            Text(job.company).font(.system(.title, design: .serif))
            Text(job.title).font(.headline)
            Text("London · Hybrid · £80–95k").font(.footnote).foregroundStyle(.secondary)
            Spacer()
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .hairlineCard()
    }
}

private struct Deck: View {
    let model: ProtoModel
    var emptyAction: AnyView?

    var body: some View {
        VStack(spacing: 20) {
            if let job = model.current {
                ProtoCard(job: job)
                    .id(job.id)
                    .swipeToDecide { dir in
                        withAnimation(.snappy) { model.decide(dir == .right ? "Shortlisted" : "Dismissed") }
                    }
                    .transition(.opacity)
                    .padding(.horizontal)
                HStack(spacing: 28) {
                    Button("Dismiss", systemImage: "xmark") { withAnimation { model.decide("Dismissed") } }
                    Menu("More", systemImage: "ellipsis") {
                        Button("Shortlist and draft CV", systemImage: "doc.badge.plus") { withAnimation { model.decide("Shortlisted, drafting CV") } }
                        Button("Exclude company", systemImage: "nosign", role: .destructive) { withAnimation { model.decide("Excluded company") } }
                        Divider()
                        if job.kind.isAgency {
                            Button("Not an agency", systemImage: "building.2") { withAnimation { model.reclassify() } }
                        } else {
                            Button("This is an agency", systemImage: "person.2") { withAnimation { model.reclassify() } }
                        }
                    }
                    Button("Shortlist", systemImage: "checkmark") { withAnimation { model.decide("Shortlisted") } }
                }
                .labelStyle(.iconOnly)
                .font(.title2)
                .buttonStyle(.bordered)
                .buttonBorderShape(.circle)
            } else {
                ContentUnavailableView {
                    Label("\(model.stack.rawValue) clear", systemImage: "checkmark.circle")
                } actions: {
                    emptyAction
                }
            }
        }
        .padding(.bottom, 56)
        .safeAreaInset(edge: .bottom) {
            if let banner = model.banner {
                HStack {
                    Text(banner).font(.footnote.weight(.semibold)).lineLimit(2)
                    Spacer()
                    Button("Undo") { withAnimation { model.undo() } }.font(.footnote.weight(.semibold))
                }
                .padding(12)
                .hairlineCard()
                .padding(.horizontal)
                .padding(.bottom, 56)
                .transition(.move(edge: .bottom).combined(with: .opacity))
            }
        }
    }
}

private struct SegmentedVariant: View {
    @State private var model = ProtoModel()
    var body: some View {
        Deck(model: model)
            .navigationTitle("Triage")
            .navigationBarTitleDisplayMode(.inline)
            .safeAreaInset(edge: .top) {
                Picker("Stack", selection: $model.stack) {
                    Text("Direct \(model.queue(.direct).count)").tag(Stack.direct)
                    Text("Agency \(model.queue(.agency).count)").tag(Stack.agency)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
            }
    }
}

private struct TitleMenuVariant: View {
    @State private var model = ProtoModel()
    var body: some View {
        Deck(model: model)
            .navigationTitle(model.stack == .direct ? "Triage" : "Agency jobs")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarTitleMenu {
                Picker("Stack", selection: $model.stack) {
                    Label("Direct (\(model.queue(.direct).count))", systemImage: "building.2").tag(Stack.direct)
                    Label("Agency (\(model.queue(.agency).count))", systemImage: "person.2").tag(Stack.agency)
                }
            }
    }
}

private struct DetourVariant: View {
    @State private var model = ProtoModel()
    var body: some View {
        Deck(model: model, emptyAction: model.stack == .direct && model.otherCount > 0
            ? AnyView(Button("Look at \(model.otherCount) agency jobs") { withAnimation { model.stack = .agency } }.buttonStyle(.borderedProminent))
            : nil)
            .background(model.stack == .agency ? Color.orange.opacity(0.06) : .clear)
            .navigationTitle(model.stack == .direct ? "Triage" : "Agency detour")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    if model.stack == .direct {
                        Button("\(model.otherCount) agency") { withAnimation { model.stack = .agency } }
                            .font(.footnote)
                    } else {
                        Button("Back to Direct") { withAnimation { model.stack = .direct } }
                    }
                }
            }
    }
}

struct AgencyStackPrototype: View {
    @State private var variant = 0
    private let names = ["A — Segmented control", "B — Title menu", "C — Agency detour"]

    var body: some View {
        NavigationStack {
            switch variant {
            case 0: SegmentedVariant()
            case 1: TitleMenuVariant()
            default: DetourVariant()
            }
        }
        .id(variant)
        .overlay(alignment: .bottom) {
            HStack(spacing: 16) {
                Button("Previous", systemImage: "chevron.left") { variant = (variant + 2) % 3 }
                Text(names[variant]).font(.caption.monospaced())
                Button("Next", systemImage: "chevron.right") { variant = (variant + 1) % 3 }
            }
            .labelStyle(.iconOnly)
            .padding(.horizontal, 14).padding(.vertical, 8)
            .foregroundStyle(.white)
            .background(.black, in: .capsule)
            .shadow(radius: 6)
        }
    }
}

#Preview("Agency stack prototype") { AgencyStackPrototype() }
#endif
