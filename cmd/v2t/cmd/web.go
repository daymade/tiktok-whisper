package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"tiktok-whisper/web"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the embedding visualization web server",
	Long: `Start a web server that provides a stunning 3D visualization of embedding data.
	
Features:
- Interactive 3D particle system with clustering
- Real-time search and similarity analysis  
- UMAP/t-SNE/PCA dimension reduction
- Beautiful particle effects and animations
- Support for both OpenAI and Gemini embeddings

The server will be available at http://localhost:8080`,
	Run: runWebServer,
}

var (
	webPort string
)

func init() {
	rootCmd.AddCommand(webCmd)

	webCmd.Flags().StringVar(&webPort, "port", ":8080", "Port to run the web server on")
}

func runWebServer(cmd *cobra.Command, args []string) {
	log.Printf("🌟 Starting embedding visualization web server...")
	log.Printf("📊 Features: 3D visualization, clustering, real-time search")
	log.Printf("🎨 Effects: particle system, animations, interactive controls")

	// Create and start the web server
	server, err := web.NewServer(webPort)
	if err != nil {
		log.Fatalf("❌ Failed to create web server: %v", err)
	}
	defer server.Close()

	log.Printf("🚀 Server starting on port %s", webPort)
	log.Printf("🌐 Open your browser and visit: http://localhost%s", webPort)
	log.Printf("✨ Enjoy the stunning embedding visualization!")

	if err := server.Start(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
