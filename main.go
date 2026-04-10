package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type Config struct {
	RTSPurl    string
	OutputDir  string
	SegmentSec int
}

func main() {
	conf := Config{
		RTSPurl:    "rtsp://192.168.1.155:554/stream1",
		OutputDir:  "./web/",
		SegmentSec: 4,
	}
	ctx := context.Background()

	go HLSWorker(ctx, conf)

	fs := http.FileServer(http.Dir(conf.OutputDir))
	http.Handle("/stream/", http.StripPrefix("/stream/", fs))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func HLSWorker(ctx context.Context, conf Config) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping HLS worker")
			return
		default:
			log.Printf("Starting FFmpeg for: %s", conf.RTSPurl)
			cmd := exec.CommandContext(ctx, "ffmpeg",
				"-rtsp_transport", "tcp", // Force TCP to prevent "smearing" artifacts
				"-i", conf.RTSPurl,
				"-c:v", "libx264", // Transcode to H.264
				"-preset", "veryfast", // Balance CPU usage and speed
				"-pix_fmt", "yuv420p", // Ensure compatibility with all browsers
				"-g", "48", // Force keyframe interval for smoother segmenting
				"-sc_threshold", "0",
				"-c:a", "aac", // Convert audio to AAC (HLS requirement)
				"-b:a", "128k",
				"-f", "hls",
				"-hls_time", fmt.Sprintf("%d", conf.SegmentSec),
				"-hls_list_size", "5", // Keep last 5 segments in manifest
				"-hls_flags", "delete_segments+independent_segments",
				fmt.Sprintf("%s/index.m3u8", conf.OutputDir),
			)

			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				log.Printf("FFmpeg exited with error: %v. Restarting in 5s...", err)
				time.Sleep(5 * time.Second)
			}

		}
	}
}
