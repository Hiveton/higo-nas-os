package mcp

import (
	"context"
	"encoding/json"
	"net/url"

	"higoos/server-go/internal/apiclient"
)

// --- input types ------------------------------------------------------------

type VideoLibraryUpdateInput struct {
	Libraries []map[string]any `json:"libraries" jsonschema:"full replacement list of video libraries"`
}

type VideoLibraryCreateInput struct {
	Name              string   `json:"name" jsonschema:"display name of the new library"`
	Type              string   `json:"type" jsonschema:"library type: movie, series or mixed"`
	Paths             []string `json:"paths" jsonschema:"filesystem paths scanned for this library"`
	MetadataLanguage  string   `json:"metadataLanguage,omitempty" jsonschema:"preferred metadata language code"`
	AllowAdultContent bool     `json:"allowAdultContent,omitempty" jsonschema:"whether adult content is allowed"`
	AutoSubtitles     bool     `json:"autoSubtitles,omitempty" jsonschema:"whether subtitles are fetched automatically"`
	SubtitleLanguage  string   `json:"subtitleLanguage,omitempty" jsonschema:"preferred subtitle language code"`
}

type VideoLibraryDeleteInput struct {
	ID string `json:"id" jsonschema:"identifier of the library to delete"`
}

type VideoItemsInput struct {
	Q         string `json:"q,omitempty" jsonschema:"free-text search query"`
	LibraryID string `json:"libraryId,omitempty" jsonschema:"restrict results to this library id"`
	Kind      string `json:"kind,omitempty" jsonschema:"restrict results to this item kind"`
}

type VideoItemInput struct {
	ID string `json:"id" jsonschema:"identifier of the video item"`
}

type VideoItemStreamInput struct {
	ID string `json:"id" jsonschema:"identifier of the video item to stream"`
}

type VideoItemPosterInput struct {
	ID string `json:"id" jsonschema:"identifier of the video item poster"`
}

type VideoItemSubtitleInput struct {
	ID string `json:"id" jsonschema:"identifier of the video item subtitle"`
}

type VideoTaskCreateInput struct {
	ItemID  string `json:"itemId" jsonschema:"identifier of the target video item"`
	Profile string `json:"profile" jsonschema:"processing profile to apply"`
}

type VideoLiveSourceCreateInput struct {
	Name        string `json:"name" jsonschema:"display name of the live TV source"`
	URL         string `json:"url" jsonschema:"playlist URL of the live TV source"`
	UserAgent   string `json:"userAgent,omitempty" jsonschema:"custom user agent for fetching the source"`
	StreamLimit int    `json:"streamLimit,omitempty" jsonschema:"maximum concurrent streams allowed"`
}

type VideoLiveChannelsInput struct {
	SourceID string `json:"sourceId,omitempty" jsonschema:"restrict channels to this source id"`
}

type VideoLiveGuideSourceCreateInput struct {
	Name      string `json:"name" jsonschema:"display name of the EPG guide source"`
	URL       string `json:"url" jsonschema:"XMLTV guide URL"`
	UserAgent string `json:"userAgent,omitempty" jsonschema:"custom user agent for fetching the guide"`
}

type VideoLiveProgramsInput struct {
	SourceID  string `json:"sourceId,omitempty" jsonschema:"restrict programs to this source id"`
	ChannelID string `json:"channelId,omitempty" jsonschema:"restrict programs to this channel id"`
	From      string `json:"from,omitempty" jsonschema:"start of the time window (RFC3339)"`
	To        string `json:"to,omitempty" jsonschema:"end of the time window (RFC3339)"`
}

type VideoDVRSettingsUpdateInput struct {
	RecordingPath       string `json:"recordingPath" jsonschema:"default path for recordings"`
	MovieRecordingPath  string `json:"movieRecordingPath,omitempty" jsonschema:"path for movie recordings"`
	SeriesRecordingPath string `json:"seriesRecordingPath,omitempty" jsonschema:"path for series recordings"`
	PrePaddingSeconds   int    `json:"prePaddingSeconds" jsonschema:"seconds to record before the program start"`
	PostPaddingSeconds  int    `json:"postPaddingSeconds" jsonschema:"seconds to record after the program end"`
	MaxConcurrentRecord int    `json:"maxConcurrentRecord" jsonschema:"maximum concurrent recordings"`
	SaveNFO             bool   `json:"saveNfo" jsonschema:"whether to save NFO metadata files"`
	SaveImages          bool   `json:"saveImages" jsonschema:"whether to save artwork images"`
	PostProcessCommand  string `json:"postProcessCommand,omitempty" jsonschema:"command run after a recording completes"`
}

type VideoRecordingTimerCreateInput struct {
	ProgramID  string `json:"programId,omitempty" jsonschema:"EPG program id to record"`
	ChannelID  string `json:"channelId,omitempty" jsonschema:"channel id to record"`
	Name       string `json:"name,omitempty" jsonschema:"display name of the recording"`
	Overview   string `json:"overview,omitempty" jsonschema:"description of the recording"`
	StartAt    string `json:"startAt,omitempty" jsonschema:"recording start time (RFC3339)"`
	EndAt      string `json:"endAt,omitempty" jsonschema:"recording end time (RFC3339)"`
	Priority   int    `json:"priority,omitempty" jsonschema:"scheduling priority"`
	TargetPath string `json:"targetPath,omitempty" jsonschema:"output path for the recording"`
}

type VideoRecordingTimerCancelInput struct {
	ID string `json:"id" jsonschema:"identifier of the recording timer to cancel"`
}

// --- registration -----------------------------------------------------------

func registerVideo(r *registry) {
	addTool(r, "video", "higo.video.library.get",
		"Get video library settings and configured libraries.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoLibrary(ctx)
		})

	addTool(r, "video", "higo.video.library.update",
		"Replace the full set of video library settings.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoLibraryUpdateInput) (json.RawMessage, error) {
			return c.VideoLibraryUpdate(ctx, map[string]any{"libraries": in.Libraries})
		})

	addTool(r, "video", "higo.video.library.create",
		"Create a new video library.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoLibraryCreateInput) (json.RawMessage, error) {
			return c.VideoLibraryCreate(ctx, in)
		})

	addTool(r, "video", "higo.video.library.delete",
		"Delete a video library and its items.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in VideoLibraryDeleteInput) (json.RawMessage, error) {
			return c.VideoLibraryDelete(ctx, in.ID)
		})

	addTool(r, "video", "higo.video.scan",
		"Trigger a scan of the video libraries.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoScan(ctx)
		})

	addTool(r, "video", "higo.video.items.list",
		"List video items, optionally filtered by query, library or kind.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VideoItemsInput) (json.RawMessage, error) {
			return c.VideoItems(ctx, in.Q, in.LibraryID, in.Kind)
		})

	addTool(r, "video", "higo.video.items.get",
		"Get a single video item by id.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VideoItemInput) (json.RawMessage, error) {
			return c.VideoItem(ctx, in.ID)
		})

	addTool(r, "video", "higo.video.items.stream.url",
		"Get the direct stream URL for a video item.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VideoItemStreamInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/videos/items/" + url.PathEscape(in.ID) + "/stream"), nil
		})

	addTool(r, "video", "higo.video.items.poster.url",
		"Get the poster image URL for a video item.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VideoItemPosterInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/videos/items/" + url.PathEscape(in.ID) + "/poster"), nil
		})

	addTool(r, "video", "higo.video.items.subtitle.url",
		"Get the subtitle file URL for a video item.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VideoItemSubtitleInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/videos/items/" + url.PathEscape(in.ID) + "/subtitle"), nil
		})

	addTool(r, "video", "higo.video.tasks.list",
		"List video processing tasks.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoTasks(ctx)
		})

	addTool(r, "video", "higo.video.tasks.scrape",
		"Create a metadata scrape task for a video item.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoTaskCreateInput) (json.RawMessage, error) {
			return c.VideoScrapeTask(ctx, in)
		})

	addTool(r, "video", "higo.video.tasks.subtitle",
		"Create a subtitle task for a video item.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoTaskCreateInput) (json.RawMessage, error) {
			return c.VideoSubtitleTask(ctx, in)
		})

	addTool(r, "video", "higo.video.tasks.transcode",
		"Create a transcode task for a video item.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoTaskCreateInput) (json.RawMessage, error) {
			return c.VideoTranscodeTask(ctx, in)
		})

	addTool(r, "video", "higo.video.live.sources.list",
		"List live TV sources.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoLiveSources(ctx)
		})

	addTool(r, "video", "higo.video.live.sources.create",
		"Create a live TV source.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoLiveSourceCreateInput) (json.RawMessage, error) {
			return c.VideoLiveSourceCreate(ctx, in)
		})

	addTool(r, "video", "higo.video.live.channels.list",
		"List live TV channels, optionally filtered by source.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VideoLiveChannelsInput) (json.RawMessage, error) {
			return c.VideoLiveChannels(ctx, in.SourceID)
		})

	addTool(r, "video", "higo.video.live.guide-sources.list",
		"List EPG guide sources.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoLiveGuideSources(ctx)
		})

	addTool(r, "video", "higo.video.live.guide-sources.create",
		"Create an EPG guide source.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoLiveGuideSourceCreateInput) (json.RawMessage, error) {
			return c.VideoLiveGuideSourceCreate(ctx, in)
		})

	addTool(r, "video", "higo.video.live.programs.list",
		"List EPG programs, optionally filtered by source, channel or time window.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VideoLiveProgramsInput) (json.RawMessage, error) {
			return c.VideoLivePrograms(ctx, in.SourceID, in.ChannelID, in.From, in.To)
		})

	addTool(r, "video", "higo.video.live.dvr.settings.get",
		"Get DVR settings.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoLiveDVRSettings(ctx)
		})

	addTool(r, "video", "higo.video.live.dvr.settings.update",
		"Replace DVR settings.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoDVRSettingsUpdateInput) (json.RawMessage, error) {
			return c.VideoLiveDVRSettingsUpdate(ctx, in)
		})

	addTool(r, "video", "higo.video.live.recording-timers.list",
		"List DVR recording timers.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoLiveRecordingTimers(ctx)
		})

	addTool(r, "video", "higo.video.live.recording-timers.create",
		"Create a DVR recording timer.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VideoRecordingTimerCreateInput) (json.RawMessage, error) {
			return c.VideoLiveRecordingTimerCreate(ctx, in)
		})

	addTool(r, "video", "higo.video.live.recording-timers.cancel",
		"Cancel a DVR recording timer.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in VideoRecordingTimerCancelInput) (json.RawMessage, error) {
			return c.VideoLiveRecordingTimerCancel(ctx, in.ID)
		})

	addTool(r, "video", "higo.video.live.recordings.list",
		"List completed DVR recordings.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VideoLiveRecordings(ctx)
		})
}
