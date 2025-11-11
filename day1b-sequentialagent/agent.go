package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/server/restapi/services"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	outlineAgent, err := llmagent.New(llmagent.Config{
		Name:        "OutlineAgent",
		Model:       model,
		Description: "Creates the initial blog post outline.",
		Instruction: `Create a blog outline for the given topic with:
    1. A catchy headline
    2. An introduction hook
    3. 3-5 main sections with 2-3 bullet points for each
    4. A concluding thought`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "blog_outline",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	writerAgent, err := llmagent.New(llmagent.Config{
		Name:        "WriterAgent",
		Model:       model,
		Description: "Writes the full blog post based on the outline from the previous agent.",
		Instruction: `Following this outline strictly: {blog_outline}
    Write a brief, 200 to 300-word blog post with an engaging and informative tone.`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "blog_draft",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	editorAgent, err := llmagent.New(llmagent.Config{
		Name:        "EditorAgent",
		Model:       model,
		Description: "Edits and polishes the draft from the writer agent.",
		Instruction: `Edit this draft: {blog_draft}
    Your task is to polish the text by fixing any grammatical errors, improving the flow and sentence structure, and enhancing overall clarity.`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "final_blog",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	rootAgent, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "BlogPipeline",
			Description: "Root Agent for the blog pipeline.",
			SubAgents:   []agent.Agent{outlineAgent, writerAgent, editorAgent},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	config := &adk.Config{
		AgentLoader: services.NewSingleAgentLoader(rootAgent),
	}

	l := full.NewLauncher()
	err = l.Execute(ctx, config, os.Args[1:])
	if err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
