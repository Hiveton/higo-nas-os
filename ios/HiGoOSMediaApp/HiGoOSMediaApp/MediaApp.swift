import SwiftUI

enum MediaKind: String, CaseIterable, Identifiable {
    case movies = "Movies"
    case tv = "TV"
    case music = "Music"
    case photos = "Photos"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .movies: "film.fill"
        case .tv: "tv.fill"
        case .music: "music.note"
        case .photos: "photo.on.rectangle.angled"
        }
    }
}

enum MediaAppTab: String, CaseIterable, Identifiable {
    case library = "Library"
    case discover = "Discover"
    case downloads = "Downloads"
    case devices = "Devices"
    case settings = "Settings"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .library: "rectangle.stack.fill"
        case .discover: "sparkles.tv.fill"
        case .downloads: "arrow.down.circle.fill"
        case .devices: "airplayvideo"
        case .settings: "gearshape.fill"
        }
    }
}

struct MediaItem: Identifiable, Equatable {
    var id: String
    var kind: MediaKind
    var title: String
    var subtitle: String
    var duration: String
    var progress: Double
    var resolution: String
    var isOffline: Bool
    var transcodeState: String
    var subtitleTracks: [String]
    var audioTracks: [String]
    var palette: CoverPalette
}

struct MusicTrack: Identifiable, Equatable {
    var id: String
    var title: String
    var artist: String
    var album: String
    var duration: String
    var isLossless: Bool
}

struct PhotoMemory: Identifiable, Equatable {
    var id: String
    var title: String
    var subtitle: String
    var count: Int
    var location: String
    var palette: CoverPalette
}

struct DownloadItem: Identifiable, Equatable {
    var id: String
    var title: String
    var subtitle: String
    var progress: Double
    var isPaused: Bool
    var speed: String
}

struct CastTarget: Identifiable, Equatable {
    var id: String
    var name: String
    var room: String
    var isSelected: Bool
}

struct MediaNasDevice: Identifiable, Equatable {
    var id: String
    var name: String
    var location: String
    var isOnline: Bool
    var usedTB: Double
    var totalTB: Double
    var remoteAccess: Bool

    var usedRatio: Double { min(max(usedTB / totalTB, 0), 1) }
    var capacityLabel: String { String(format: "%.1f / %.0f TB", usedTB, totalTB) }
}

struct PlaybackState: Equatable {
    var isPlaying = false
    var progress = 0.34
    var speed = "1.0x"
    var subtitleTrack = "English"
    var audioTrack = "Dolby Atmos"
    var selectedCastTargetID: String?

    mutating func togglePlayPause() {
        isPlaying.toggle()
    }
}

struct CoverPalette: Equatable {
    var start: Color
    var end: Color
    var symbol: String
}

protocol MediaRepository {
    func mediaItems() -> [MediaItem]
    func musicTracks() -> [MusicTrack]
    func photoMemories() -> [PhotoMemory]
    func downloads() -> [DownloadItem]
    func castTargets() -> [CastTarget]
    func nasDevice() -> MediaNasDevice
}

struct StaticMediaRepository: MediaRepository {
    func mediaItems() -> [MediaItem] {
        [
            MediaItem(
                id: "movie-sea",
                kind: .movies,
                title: "Deep Sea Archive",
                subtitle: "4K HDR movie stored on HiGoOS",
                duration: "2h 12m",
                progress: 0.68,
                resolution: "4K HDR",
                isOffline: true,
                transcodeState: "Direct Play",
                subtitleTracks: ["English", "Chinese", "Japanese"],
                audioTracks: ["Dolby Atmos", "AAC Stereo"],
                palette: .init(start: .cyan, end: .indigo, symbol: "water.waves")
            ),
            MediaItem(
                id: "movie-city",
                kind: .movies,
                title: "City Lights Backup",
                subtitle: "Family cinema collection",
                duration: "1h 48m",
                progress: 0.12,
                resolution: "1080P",
                isOffline: false,
                transcodeState: "Ready",
                subtitleTracks: ["English", "Chinese"],
                audioTracks: ["AAC Stereo"],
                palette: .init(start: .orange, end: .pink, symbol: "building.2.crop.circle")
            ),
            MediaItem(
                id: "tv-workshop",
                kind: .tv,
                title: "NAS Workshop S02E04",
                subtitle: "Smart home streaming tips",
                duration: "45m",
                progress: 0.42,
                resolution: "4K",
                isOffline: false,
                transcodeState: "Transcoding",
                subtitleTracks: ["English"],
                audioTracks: ["AAC Stereo", "Commentary"],
                palette: .init(start: .green, end: .teal, symbol: "wrench.and.screwdriver.fill")
            ),
            MediaItem(
                id: "tv-food",
                kind: .tv,
                title: "Weekend Kitchen E08",
                subtitle: "Shared family library",
                duration: "38m",
                progress: 0.0,
                resolution: "1080P",
                isOffline: true,
                transcodeState: "Direct Play",
                subtitleTracks: ["Chinese"],
                audioTracks: ["AAC Stereo"],
                palette: .init(start: .yellow, end: .red, symbol: "fork.knife")
            )
        ]
    }

    func musicTracks() -> [MusicTrack] {
        [
            MusicTrack(id: "tr-1", title: "Harbor Night", artist: "HiGo Sessions", album: "NAS Mix", duration: "3:48", isLossless: true),
            MusicTrack(id: "tr-2", title: "Offline Morning", artist: "Family Room", album: "Road Trip", duration: "4:16", isLossless: false),
            MusicTrack(id: "tr-3", title: "Storage Lights", artist: "Hiveton Lab", album: "Focus", duration: "2:58", isLossless: true)
        ]
    }

    func photoMemories() -> [PhotoMemory] {
        [
            PhotoMemory(id: "pm-1", title: "Summer Family Roll", subtitle: "128 photos synced from phones", count: 128, location: "Shenzhen", palette: .init(start: .mint, end: .blue, symbol: "sun.max.fill")),
            PhotoMemory(id: "pm-2", title: "Workshop Week", subtitle: "Auto grouped by HiGoOS Photos", count: 46, location: "Lab", palette: .init(start: .purple, end: .blue, symbol: "camera.macro")),
            PhotoMemory(id: "pm-3", title: "Travel Highlights", subtitle: "Offline preview ready", count: 87, location: "Xiamen", palette: .init(start: .orange, end: .cyan, symbol: "map.fill"))
        ]
    }

    func downloads() -> [DownloadItem] {
        [
            DownloadItem(id: "dl-1", title: "Deep Sea Archive", subtitle: "4K HDR movie", progress: 0.74, isPaused: false, speed: "12.4 MB/s"),
            DownloadItem(id: "dl-2", title: "NAS Workshop S02", subtitle: "6 episodes", progress: 0.38, isPaused: true, speed: "Paused"),
            DownloadItem(id: "dl-3", title: "Summer Family Roll", subtitle: "Photo memory", progress: 1.0, isPaused: false, speed: "Ready offline")
        ]
    }

    func castTargets() -> [CastTarget] {
        [
            CastTarget(id: "living", name: "Living Room TV", room: "AirPlay", isSelected: true),
            CastTarget(id: "studio", name: "Studio Display", room: "HiGo Cast", isSelected: false),
            CastTarget(id: "bedroom", name: "Bedroom Projector", room: "DLNA", isSelected: false)
        ]
    }

    func nasDevice() -> MediaNasDevice {
        MediaNasDevice(id: "higoos-home", name: "HiGoOS Home NAS", location: "LAN + Remote", isOnline: true, usedTB: 7.4, totalTB: 12, remoteAccess: true)
    }
}

@MainActor
final class MediaAppViewModel: ObservableObject {
    @Published var selectedTab: MediaAppTab = .library
    @Published var selectedKind: MediaKind = .movies
    @Published var searchText = ""
    @Published var selectedMedia: MediaItem?
    @Published var selectedTrack: MusicTrack?
    @Published var selectedMemory: PhotoMemory?
    @Published var playback = PlaybackState()
    @Published var nasDevice: MediaNasDevice
    @Published var downloads: [DownloadItem]
    @Published var castTargets: [CastTarget]

    let mediaItems: [MediaItem]
    let musicTracks: [MusicTrack]
    let photoMemories: [PhotoMemory]

    init(repository: MediaRepository) {
        mediaItems = repository.mediaItems()
        musicTracks = repository.musicTracks()
        photoMemories = repository.photoMemories()
        downloads = repository.downloads()
        castTargets = repository.castTargets()
        nasDevice = repository.nasDevice()
        selectedMedia = mediaItems.first
        selectedTrack = musicTracks.first
        selectedMemory = photoMemories.first
        playback.selectedCastTargetID = castTargets.first(where: \.isSelected)?.id
    }

    var filteredMedia: [MediaItem] {
        let items = mediaItems.filter { $0.kind == selectedKind }
        guard !searchText.isEmpty else { return items }
        return items.filter { $0.title.localizedCaseInsensitiveContains(searchText) || $0.subtitle.localizedCaseInsensitiveContains(searchText) }
    }

    func selectKind(_ kind: MediaKind) {
        selectedKind = kind
        if let first = filteredMedia.first { selectedMedia = first }
    }

    func selectMedia(_ item: MediaItem) {
        selectedMedia = item
        playback.isPlaying = true
    }

    func selectTrack(_ track: MusicTrack) {
        selectedTrack = track
        playback.isPlaying = true
    }

    func selectMemory(_ memory: PhotoMemory) {
        selectedMemory = memory
    }

    func toggleDownload(_ id: String) {
        guard let index = downloads.firstIndex(where: { $0.id == id }) else { return }
        downloads[index].isPaused.toggle()
        downloads[index].speed = downloads[index].isPaused ? "Paused" : "9.8 MB/s"
    }

    func selectCastTarget(_ id: String) {
        playback.selectedCastTargetID = id
        castTargets = castTargets.map { CastTarget(id: $0.id, name: $0.name, room: $0.room, isSelected: $0.id == id) }
    }

    func toggleNasOnline() {
        nasDevice.isOnline.toggle()
    }

    func simulateCapacityChange() {
        nasDevice.usedTB = nasDevice.usedTB > 8.5 ? 6.8 : 9.1
    }
}

extension Color {
    static let mediaTeal = Color(red: 0.0, green: 0.45, blue: 0.39)
    static let mediaInk = Color(red: 0.05, green: 0.07, blue: 0.09)
    static let mediaSurface = Color(red: 0.95, green: 0.97, blue: 0.97)
    static let mediaLine = Color.black.opacity(0.08)
}

struct MediaRootView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    var body: some View {
        if horizontalSizeClass == .regular {
            MediaPadRootView()
        } else {
            MediaPhoneRootView()
        }
    }
}

struct MediaPhoneRootView: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @State private var showingPlayer = false

    var body: some View {
        VStack(spacing: 0) {
            MediaPhoneHeader()
                .padding(.horizontal, 18)
                .padding(.top, 12)
                .padding(.bottom, 10)

            Group {
                switch model.selectedTab {
                case .library:
                    LibraryDashboard(showingPlayer: $showingPlayer)
                case .discover:
                    DiscoverDashboard()
                case .downloads:
                    DownloadsDashboard()
                case .devices:
                    DevicesDashboard()
                case .settings:
                    SettingsDashboard()
                }
            }

            VStack(spacing: 10) {
                MiniPlayerBar(showingPlayer: $showingPlayer)
                MediaBottomDock()
            }
            .padding(.horizontal, 14)
            .padding(.top, 10)
            .padding(.bottom, 10)
            .background(.regularMaterial)
        }
        .background(Color.mediaSurface.ignoresSafeArea())
        .sheet(isPresented: $showingPlayer) {
            FullPlayerView()
        }
    }
}

struct MediaPhoneHeader: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        VStack(spacing: 12) {
            HStack(alignment: .center) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("HiGoOS Media")
                        .font(.title2.weight(.bold))
                    Text(model.nasDevice.name)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button {
                    model.playback.togglePlayPause()
                } label: {
                    Image(systemName: model.playback.isPlaying ? "pause.fill" : "play.fill")
                        .font(.headline)
                        .frame(width: 44, height: 44)
                        .background(Color.mediaTeal, in: Circle())
                        .foregroundStyle(.white)
                }
                .accessibilityLabel(model.playback.isPlaying ? "Pause" : "Play")
            }

            NasCompactStatus()
        }
    }
}

struct NasCompactStatus: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: model.nasDevice.isOnline ? "checkmark.icloud.fill" : "icloud.slash.fill")
                .foregroundStyle(model.nasDevice.isOnline ? Color.mediaTeal : .red)
                .frame(width: 34, height: 34)
                .background(.white, in: RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    Text(model.nasDevice.isOnline ? "Online" : "Offline")
                        .font(.subheadline.weight(.semibold))
                    Text(model.nasDevice.location)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Spacer()
                    Text(model.nasDevice.capacityLabel)
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(.secondary)
                }
                ProgressView(value: model.nasDevice.usedRatio)
                    .tint(.mediaTeal)
            }
        }
        .padding(12)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct LibraryDashboard: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Binding var showingPlayer: Bool

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                SearchBar(text: $model.searchText)
                MediaKindStrip()

                if model.selectedKind == .music {
                    MusicSection()
                } else if model.selectedKind == .photos {
                    PhotoSection()
                } else {
                    ContinueWatchingCard(showingPlayer: $showingPlayer)
                    PosterGrid(showingPlayer: $showingPlayer)
                }
            }
            .padding(.horizontal, 18)
            .padding(.vertical, 14)
        }
    }
}

struct SearchBar: View {
    @Binding var text: String

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(.secondary)
            TextField("Search movies, music, photos", text: $text)
                .textInputAutocapitalization(.never)
            if !text.isEmpty {
                Button {
                    text = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                }
                .accessibilityLabel("Clear search")
            }
        }
        .padding(12)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct MediaKindStrip: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(MediaKind.allCases) { kind in
                    Button {
                        model.selectKind(kind)
                    } label: {
                        Label(kind.rawValue, systemImage: kind.icon)
                            .font(.subheadline.weight(.semibold))
                            .padding(.horizontal, 12)
                            .frame(height: 38)
                            .background(model.selectedKind == kind ? Color.mediaTeal : .white, in: Capsule())
                            .foregroundStyle(model.selectedKind == kind ? .white : .primary)
                            .overlay(Capsule().stroke(Color.mediaLine))
                    }
                    .accessibilityIdentifier("kind-\(kind.rawValue.lowercased())")
                }
            }
        }
    }
}

struct ContinueWatchingCard: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Binding var showingPlayer: Bool

    var body: some View {
        if let item = model.selectedMedia ?? model.filteredMedia.first {
            Button {
                model.selectMedia(item)
                showingPlayer = true
            } label: {
                HStack(spacing: 14) {
                    CoverArt(palette: item.palette, title: item.title, height: 154)
                        .frame(width: 118)

                    VStack(alignment: .leading, spacing: 10) {
                        Text("Continue Watching")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(Color.mediaTeal)
                        Text(item.title)
                            .font(.title3.weight(.bold))
                            .foregroundStyle(.primary)
                            .lineLimit(2)
                        Text("\(item.resolution) - \(item.transcodeState)")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        ProgressView(value: item.progress)
                            .tint(.mediaTeal)
                        HStack {
                            Label(item.duration, systemImage: "clock")
                            if item.isOffline {
                                Label("Offline", systemImage: "arrow.down.circle.fill")
                            }
                        }
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    }
                    Spacer(minLength: 0)
                }
                .padding(12)
                .background(.white, in: RoundedRectangle(cornerRadius: 8))
                .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
            }
            .buttonStyle(.plain)
            .accessibilityIdentifier("continue-watching-card")
        }
    }
}

struct PosterGrid: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Binding var showingPlayer: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            SectionHeader(title: model.selectedKind == .tv ? "TV on NAS" : "Movies on NAS", action: "Sort")

            LazyVGrid(columns: [GridItem(.adaptive(minimum: 150), spacing: 12)], spacing: 12) {
                ForEach(model.filteredMedia) { item in
                    Button {
                        model.selectMedia(item)
                        showingPlayer = true
                    } label: {
                        PosterTile(item: item)
                    }
                    .buttonStyle(.plain)
                    .accessibilityIdentifier("media-card-\(item.id)")
                }
            }
        }
    }
}

struct PosterTile: View {
    var item: MediaItem

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CoverArt(palette: item.palette, title: item.title, height: 138)
            Text(item.title)
                .font(.headline)
                .lineLimit(2)
            Text(item.subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(2)
            HStack {
                Text(item.duration)
                Spacer()
                if item.isOffline {
                    Image(systemName: "arrow.down.circle.fill")
                        .foregroundStyle(Color.mediaTeal)
                }
            }
            .font(.caption2)
            .foregroundStyle(.secondary)
        }
        .padding(10)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct MusicSection: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @State private var showingPlayer = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            SectionHeader(title: "HiGoOS Music", action: "Lossless")
            ForEach(model.musicTracks) { track in
                Button {
                    model.selectTrack(track)
                    showingPlayer = true
                } label: {
                    MusicRow(track: track)
                }
                .buttonStyle(.plain)
                .accessibilityIdentifier("music-track-\(track.id)")
            }
        }
        .sheet(isPresented: $showingPlayer) {
            MusicPlayerView()
        }
    }
}

struct MusicRow: View {
    var track: MusicTrack

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: track.isLossless ? "waveform.circle.fill" : "music.note")
                .font(.title2)
                .foregroundStyle(Color.mediaTeal)
                .frame(width: 42, height: 42)
                .background(Color.mediaSurface, in: RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 3) {
                Text(track.title)
                    .font(.headline)
                Text("\(track.artist) - \(track.album)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Text(track.duration)
                .font(.caption.monospacedDigit())
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct PhotoSection: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @State private var showingMemory = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            SectionHeader(title: "Photo Memories", action: "Albums")
            ForEach(model.photoMemories) { memory in
                Button {
                    model.selectMemory(memory)
                    showingMemory = true
                } label: {
                    HStack(spacing: 12) {
                        CoverArt(palette: memory.palette, title: memory.title, height: 96)
                            .frame(width: 110)
                        VStack(alignment: .leading, spacing: 6) {
                            Text(memory.title)
                                .font(.headline)
                            Text(memory.subtitle)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .lineLimit(2)
                            Label("\(memory.count) photos - \(memory.location)", systemImage: "photo.stack")
                                .font(.caption)
                                .foregroundStyle(Color.mediaTeal)
                        }
                        Spacer(minLength: 0)
                    }
                    .padding(10)
                    .background(.white, in: RoundedRectangle(cornerRadius: 8))
                    .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
                }
                .buttonStyle(.plain)
                .accessibilityIdentifier("photo-memory-\(memory.id)")
            }
        }
        .sheet(isPresented: $showingMemory) {
            PhotoMemoryView()
        }
    }
}

struct DiscoverDashboard: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                SectionHeader(title: "Discover", action: "Local")
                InsightCard(title: "Ready for Direct Play", value: "2 videos", icon: "bolt.fill", color: .mediaTeal)
                InsightCard(title: "Recently indexed", value: "174 items", icon: "sparkles", color: .purple)
                InsightCard(title: "Family evening queue", value: "5 picks", icon: "person.2.fill", color: .orange)
            }
            .padding(18)
        }
    }
}

struct DownloadsDashboard: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                SectionHeader(title: "Offline Downloads", action: "\(model.downloads.count)")
                ForEach(model.downloads) { item in
                    DownloadRow(item: item)
                }
            }
            .padding(18)
        }
    }
}

struct DownloadRow: View {
    @EnvironmentObject private var model: MediaAppViewModel
    var item: DownloadItem

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(item.title)
                        .font(.headline)
                    Text(item.subtitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button(item.isPaused ? "Resume" : "Pause") {
                    model.toggleDownload(item.id)
                }
                .buttonStyle(.bordered)
                .accessibilityIdentifier("download-toggle-\(item.id)")
            }
            ProgressView(value: item.progress)
                .tint(item.isPaused ? .orange : .mediaTeal)
            Text(item.speed)
                .font(.caption.monospacedDigit())
                .foregroundStyle(.secondary)
        }
        .padding(14)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct DevicesDashboard: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                SectionHeader(title: "Devices", action: "Cast")
                NasCompactStatus()
                ForEach(model.castTargets) { target in
                    Button {
                        model.selectCastTarget(target.id)
                    } label: {
                        HStack {
                            Label(target.name, systemImage: target.isSelected ? "checkmark.circle.fill" : "airplayvideo")
                            Spacer()
                            Text(target.room)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .padding(14)
                        .background(.white, in: RoundedRectangle(cornerRadius: 8))
                        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
                    }
                    .buttonStyle(.plain)
                }
                Button("Simulate Capacity Change") {
                    model.simulateCapacityChange()
                }
                .buttonStyle(.borderedProminent)
                .accessibilityIdentifier("capacity-toggle-button")
            }
            .padding(18)
        }
    }
}

struct SettingsDashboard: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                SectionHeader(title: "Settings", action: "iOS")
                SettingsRow(title: "Streaming Quality", value: "Auto 4K / Direct Play", icon: "slider.horizontal.3")
                SettingsRow(title: "Remote Access", value: "Enabled", icon: "network")
                SettingsRow(title: "Transcoding", value: "Hardware preferred", icon: "cpu")
                SettingsRow(title: "Offline Cache", value: "Wi-Fi only", icon: "arrow.down.circle")
            }
            .padding(18)
        }
    }
}

struct MiniPlayerBar: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Binding var showingPlayer: Bool

    var body: some View {
        Button {
            showingPlayer = true
        } label: {
            HStack(spacing: 10) {
                CoverArt(palette: model.selectedMedia?.palette ?? .init(start: .cyan, end: .indigo, symbol: "play.rectangle.fill"), title: model.selectedMedia?.title ?? "Now Playing", height: 46)
                    .frame(width: 46)
                VStack(alignment: .leading, spacing: 2) {
                    Text(model.selectedMedia?.title ?? model.selectedTrack?.title ?? "Deep Sea Archive")
                        .font(.subheadline.weight(.semibold))
                        .lineLimit(1)
                    Text(model.playback.isPlaying ? "Playing from NAS" : "Ready to play")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button {
                    model.playback.togglePlayPause()
                } label: {
                    Image(systemName: model.playback.isPlaying ? "pause.fill" : "play.fill")
                        .frame(width: 34, height: 34)
                        .background(Color.mediaTeal, in: Circle())
                        .foregroundStyle(.white)
                }
                .accessibilityLabel(model.playback.isPlaying ? "Pause" : "Play")
            }
            .padding(10)
            .background(.white, in: RoundedRectangle(cornerRadius: 8))
            .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
        }
        .buttonStyle(.plain)
    }
}

struct MediaBottomDock: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        HStack(spacing: 0) {
            ForEach(MediaAppTab.allCases) { tab in
                Button {
                    model.selectedTab = tab
                } label: {
                    VStack(spacing: 4) {
                        Image(systemName: tab.icon)
                            .font(.system(size: 18, weight: .semibold))
                        Text(tab.rawValue)
                            .font(.caption2.weight(.semibold))
                            .lineLimit(1)
                            .minimumScaleFactor(0.7)
                    }
                    .frame(maxWidth: .infinity, minHeight: 52)
                    .foregroundStyle(model.selectedTab == tab ? Color.mediaTeal : Color.secondary)
                }
                .accessibilityIdentifier("nav-\(tab.rawValue.lowercased())")
            }
        }
        .padding(4)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct MediaPadRootView: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @State private var showingPlayer = false

    var body: some View {
        GeometryReader { proxy in
            let sidebarWidth = min(max(proxy.size.width * 0.25, 190), 230)
            let libraryWidth = min(max(proxy.size.width * 0.38, 310), 360)

            HStack(spacing: 0) {
                PadSidebar()
                    .frame(width: sidebarWidth)
                Divider()
                PadLibraryColumn(showingPlayer: $showingPlayer)
                    .frame(width: libraryWidth)
                Divider()
                PadDetailColumn(showingPlayer: $showingPlayer)
                    .frame(maxWidth: .infinity)
            }
        }
        .background(Color.mediaSurface.ignoresSafeArea())
        .sheet(isPresented: $showingPlayer) {
            FullPlayerView()
        }
    }
}

struct PadSidebar: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("HiGoOS Media")
                .font(.title2.weight(.bold))
            NasCompactStatus()

            VStack(alignment: .leading, spacing: 8) {
                Text("Library")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                ForEach(MediaKind.allCases) { kind in
                    SidebarButton(title: kind.rawValue, icon: kind.icon, selected: model.selectedTab == .library && model.selectedKind == kind) {
                        model.selectedTab = .library
                        model.selectKind(kind)
                    }
                }
            }

            VStack(alignment: .leading, spacing: 8) {
                Text("Spaces")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                ForEach(MediaAppTab.allCases.filter { $0 != .library }) { tab in
                    SidebarButton(title: tab.rawValue, icon: tab.icon, selected: model.selectedTab == tab) {
                        model.selectedTab = tab
                    }
                }
            }
            Spacer()
        }
        .padding(.top, 18)
        .padding(.bottom, 18)
        .padding(.leading, 22)
        .padding(.trailing, 12)
    }
}

struct SidebarButton: View {
    var title: String
    var icon: String
    var selected: Bool
    var action: () -> Void

    var body: some View {
        Button(action: action) {
            Label(title, systemImage: icon)
                .font(.subheadline.weight(.semibold))
                .frame(maxWidth: .infinity, minHeight: 40, alignment: .leading)
                .padding(.horizontal, 10)
                .background(selected ? Color.mediaTeal.opacity(0.13) : Color.clear, in: RoundedRectangle(cornerRadius: 8))
                .foregroundStyle(selected ? Color.mediaTeal : Color.primary)
        }
        .buttonStyle(.plain)
    }
}

struct PadLibraryColumn: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Binding var showingPlayer: Bool

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                SearchBar(text: $model.searchText)
                switch model.selectedTab {
                case .library:
                    MediaKindStrip()
                    if model.selectedKind == .music {
                        MusicSection()
                    } else if model.selectedKind == .photos {
                        PhotoSection()
                    } else {
                        PosterGrid(showingPlayer: $showingPlayer)
                    }
                case .discover:
                    DiscoverDashboard()
                case .downloads:
                    DownloadsDashboard()
                case .devices:
                    DevicesDashboard()
                case .settings:
                    SettingsDashboard()
                }
            }
            .padding(18)
        }
    }
}

struct PadDetailColumn: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Binding var showingPlayer: Bool

    var body: some View {
        VStack(spacing: 18) {
            if model.selectedKind == .music {
                MusicNowPlayingPanel()
            } else if model.selectedKind == .photos {
                PhotoMemoryPanel()
            } else if let item = model.selectedMedia {
                DetailPlayerPanel(item: item, showingPlayer: $showingPlayer)
            }
            Spacer()
        }
        .padding(22)
        .background(Color.mediaInk)
    }
}

struct DetailPlayerPanel: View {
    @EnvironmentObject private var model: MediaAppViewModel
    var item: MediaItem
    @Binding var showingPlayer: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            CoverArt(palette: item.palette, title: item.title, height: 280)
            Text(item.title)
                .font(.largeTitle.weight(.bold))
                .foregroundStyle(.white)
            Text(item.subtitle)
                .foregroundStyle(.white.opacity(0.66))
            PlayerControlBlock()
            Button {
                showingPlayer = true
            } label: {
                Label("Open Player", systemImage: "play.rectangle.fill")
                    .frame(maxWidth: .infinity, minHeight: 46)
            }
            .buttonStyle(.borderedProminent)
            .tint(.mediaTeal)
            MediaNowPlayingNasCard()
        }
    }
}

struct FullPlayerView: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var activeSheet: PlayerSheet?
    @State private var showingCast = false

    var body: some View {
        NavigationStack {
            ZStack {
                Color.mediaInk.ignoresSafeArea()
                VStack(spacing: 22) {
                    CoverArt(palette: model.selectedMedia?.palette ?? .init(start: .cyan, end: .indigo, symbol: "play.rectangle.fill"), title: model.selectedMedia?.title ?? "Now Playing", height: 310)
                    VStack(spacing: 6) {
                        Text(model.selectedMedia?.title ?? "Deep Sea Archive")
                            .font(.title.weight(.bold))
                            .foregroundStyle(.white)
                        Text(model.selectedMedia?.subtitle ?? "HiGoOS media")
                            .foregroundStyle(.white.opacity(0.66))
                    }
                    PlayerControlBlock()
                    HStack(spacing: 10) {
                        PlayerOptionButton(title: model.playback.subtitleTrack, icon: "captions.bubble") { activeSheet = .subtitles }
                        PlayerOptionButton(title: model.playback.audioTrack, icon: "waveform") { activeSheet = .audio }
                        PlayerOptionButton(title: model.playback.speed, icon: "speedometer") { activeSheet = .speed }
                        PlayerOptionButton(title: "Cast", icon: "airplayvideo") { showingCast = true }
                    }
                    MediaNowPlayingNasCard()
                    Spacer(minLength: 0)
                }
                .padding()
            }
            .navigationTitle("Now Playing")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Done") { dismiss() }
                        .foregroundStyle(.white)
                }
            }
            .sheet(item: $activeSheet) { PlayerSheetView(sheet: $0) }
            .sheet(isPresented: $showingCast) { CastTargetSheet() }
        }
    }
}

struct PlayerControlBlock: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        VStack(spacing: 14) {
            Slider(value: $model.playback.progress, in: 0...1)
                .tint(.mediaTeal)
            HStack {
                Text("24:12")
                Spacer()
                Text(model.selectedMedia?.duration ?? "2h 12m")
            }
            .font(.caption.monospacedDigit())
            .foregroundStyle(.white.opacity(0.6))
            HStack(spacing: 30) {
                Button {
                    model.playback.progress = max(0, model.playback.progress - 0.1)
                } label: {
                    Image(systemName: "gobackward.10")
                }
                Button {
                    model.playback.togglePlayPause()
                } label: {
                    Image(systemName: model.playback.isPlaying ? "pause.circle.fill" : "play.circle.fill")
                        .font(.system(size: 58))
                }
                .accessibilityIdentifier("play-pause-button")
                Button {
                    model.playback.progress = min(1, model.playback.progress + 0.1)
                } label: {
                    Image(systemName: "goforward.10")
                }
            }
            .font(.title2)
            .foregroundStyle(.white)
        }
    }
}

enum PlayerSheet: String, Identifiable {
    case subtitles = "Subtitles"
    case audio = "Audio"
    case speed = "Speed"
    var id: String { rawValue }
}

struct PlayerOptionButton: View {
    var title: String
    var icon: String
    var action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 6) {
                Image(systemName: icon)
                Text(title)
                    .font(.caption2)
                    .lineLimit(1)
                    .minimumScaleFactor(0.75)
            }
            .frame(maxWidth: .infinity, minHeight: 56)
            .background(.white.opacity(0.1), in: RoundedRectangle(cornerRadius: 8))
            .foregroundStyle(.white)
        }
        .buttonStyle(.plain)
    }
}

struct PlayerSheetView: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Environment(\.dismiss) private var dismiss
    var sheet: PlayerSheet

    var options: [String] {
        switch sheet {
        case .subtitles:
            model.selectedMedia?.subtitleTracks ?? ["English", "Chinese"]
        case .audio:
            model.selectedMedia?.audioTracks ?? ["Dolby Atmos", "AAC Stereo"]
        case .speed:
            ["0.75x", "1.0x", "1.25x", "1.5x", "2.0x"]
        }
    }

    var body: some View {
        NavigationStack {
            List(options, id: \.self) { option in
                Button {
                    switch sheet {
                    case .subtitles: model.playback.subtitleTrack = option
                    case .audio: model.playback.audioTrack = option
                    case .speed: model.playback.speed = option
                    }
                    dismiss()
                } label: {
                    HStack {
                        Text(option)
                        Spacer()
                        if selected(option) {
                            Image(systemName: "checkmark")
                                .foregroundStyle(Color.mediaTeal)
                        }
                    }
                }
            }
            .navigationTitle(sheet.rawValue)
        }
        .presentationDetents([.medium])
    }

    private func selected(_ option: String) -> Bool {
        switch sheet {
        case .subtitles: model.playback.subtitleTrack == option
        case .audio: model.playback.audioTrack == option
        case .speed: model.playback.speed == option
        }
    }
}

struct CastTargetSheet: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List(model.castTargets) { target in
                Button {
                    model.selectCastTarget(target.id)
                    dismiss()
                } label: {
                    HStack {
                        Label(target.name, systemImage: target.isSelected ? "checkmark.circle.fill" : "airplayvideo")
                        Spacer()
                        Text(target.room)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .navigationTitle("Cast Target")
        }
        .presentationDetents([.medium])
    }
}

struct MediaNowPlayingNasCard: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Label(model.nasDevice.name, systemImage: "externaldrive.connected.to.line.below")
                Spacer()
                Text(model.selectedMedia?.transcodeState ?? "Direct Play")
            }
            ProgressView(value: model.nasDevice.usedRatio)
                .tint(.mediaTeal)
            Text("NAS capacity \(model.nasDevice.capacityLabel)")
                .font(.caption)
                .foregroundStyle(.white.opacity(0.62))
        }
        .font(.caption)
        .foregroundStyle(.white)
        .padding()
        .background(.white.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
    }
}

struct MusicPlayerView: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 22) {
                CoverArt(palette: .init(start: .purple, end: .mediaTeal, symbol: "waveform"), title: model.selectedTrack?.title ?? "Harbor Night", height: 280)
                Text(model.selectedTrack?.title ?? "Harbor Night")
                    .font(.title2.weight(.bold))
                Text(model.selectedTrack?.artist ?? "HiGo Sessions")
                    .foregroundStyle(.secondary)
                PlayerControlBlock()
                    .colorScheme(.dark)
                    .padding()
                    .background(Color.mediaInk, in: RoundedRectangle(cornerRadius: 8))
                Spacer()
            }
            .padding()
            .background(Color.mediaSurface)
            .navigationTitle("Now Playing")
            .toolbar { ToolbarItem(placement: .topBarLeading) { Button("Done") { dismiss() } } }
        }
    }
}

struct MusicNowPlayingPanel: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        VStack(spacing: 18) {
            CoverArt(palette: .init(start: .purple, end: .mediaTeal, symbol: "waveform"), title: model.selectedTrack?.title ?? "Harbor Night", height: 260)
            Text(model.selectedTrack?.title ?? "Harbor Night")
                .font(.largeTitle.weight(.bold))
                .foregroundStyle(.white)
            Text(model.selectedTrack?.artist ?? "HiGo Sessions")
                .foregroundStyle(.white.opacity(0.66))
            PlayerControlBlock()
        }
    }
}

struct PhotoMemoryView: View {
    @EnvironmentObject private var model: MediaAppViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    CoverArt(palette: model.selectedMemory?.palette ?? .init(start: .mint, end: .blue, symbol: "photo.stack"), title: model.selectedMemory?.title ?? "Summer Family Roll", height: 300)
                    Text(model.selectedMemory?.title ?? "Summer Family Roll")
                        .font(.largeTitle.weight(.bold))
                    Text(model.selectedMemory?.subtitle ?? "Photos synced from HiGoOS")
                        .foregroundStyle(.secondary)
                    HStack(spacing: 12) {
                        PhotoMemoryStat(value: "\(model.selectedMemory?.count ?? 128)", title: "Photos")
                        PhotoMemoryStat(value: model.selectedMemory?.location ?? "Shenzhen", title: "Location")
                        PhotoMemoryStat(value: "Offline", title: "Preview")
                    }
                }
                .padding()
            }
            .navigationTitle("Memory")
            .toolbar { ToolbarItem(placement: .topBarLeading) { Button("Done") { dismiss() } } }
        }
    }
}

struct PhotoMemoryPanel: View {
    @EnvironmentObject private var model: MediaAppViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            CoverArt(palette: model.selectedMemory?.palette ?? .init(start: .mint, end: .blue, symbol: "photo.stack"), title: model.selectedMemory?.title ?? "Summer Family Roll", height: 300)
            Text(model.selectedMemory?.title ?? "Summer Family Roll")
                .font(.largeTitle.weight(.bold))
                .foregroundStyle(.white)
            Text(model.selectedMemory?.subtitle ?? "Photos synced from HiGoOS")
                .foregroundStyle(.white.opacity(0.66))
        }
    }
}

struct PhotoMemoryStat: View {
    var value: String
    var title: String

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(value)
                .font(.headline)
                .lineLimit(1)
                .minimumScaleFactor(0.7)
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct CoverArt: View {
    var palette: CoverPalette
    var title: String
    var height: CGFloat

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            LinearGradient(colors: [palette.start, palette.end], startPoint: .topLeading, endPoint: .bottomTrailing)
            Image(systemName: palette.symbol)
                .font(.system(size: height * 0.36, weight: .semibold))
                .foregroundStyle(.white.opacity(0.26))
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(height > 180 ? .title3.weight(.bold) : .caption.weight(.bold))
                    .foregroundStyle(.white)
                    .lineLimit(2)
                Text("HiGoOS NAS")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(.white.opacity(0.72))
            }
            .padding(12)
        }
        .frame(maxWidth: .infinity, minHeight: height, maxHeight: height)
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}

struct SectionHeader: View {
    var title: String
    var action: String

    var body: some View {
        HStack {
            Text(title)
                .font(.headline)
            Spacer()
            Text(action)
                .font(.caption.weight(.semibold))
                .foregroundStyle(Color.mediaTeal)
        }
    }
}

struct InsightCard: View {
    var title: String
    var value: String
    var icon: String
    var color: Color

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: icon)
                .foregroundStyle(.white)
                .frame(width: 44, height: 44)
                .background(color, in: RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 3) {
                Text(title)
                    .font(.headline)
                Text(value)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
        }
        .padding(14)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

struct SettingsRow: View {
    var title: String
    var value: String
    var icon: String

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: icon)
                .foregroundStyle(Color.mediaTeal)
                .frame(width: 38, height: 38)
                .background(Color.mediaSurface, in: RoundedRectangle(cornerRadius: 8))
            Text(title)
            Spacer()
            Text(value)
                .font(.caption)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.trailing)
        }
        .padding(14)
        .background(.white, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.mediaLine))
    }
}

#Preview("HiGoOSMediaApp") {
    MediaRootView()
        .environmentObject(MediaAppViewModel(repository: StaticMediaRepository()))
}
