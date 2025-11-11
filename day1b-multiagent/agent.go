package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/server/restapi/services"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/agenttool"
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

	researchAgent, err := llmagent.New(llmagent.Config{
		Name:        "ResearchAgent",
		Model:       model,
		Description: "Its job is to use the google_search tool and present findings",
		Instruction: `You are a specialized research agent. Your only job is to use the
		google_search tool to find 2-3 pieces of relevant information on the given topic and present the findings with citations.`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "research_findings",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	summarizerAgent, err := llmagent.New(llmagent.Config{
		Name:        "SummarizerAgent",
		Model:       model,
		Description: "Its job is to summarize the text it receives.",
		Instruction: `Read the provided research findings: {research_findings}
Create a concise summary as a bulleted list with 3-5 key points.`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "final_summary",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "ResearchCoordinator",
		Model:       model,
		Description: "Orchestrates the workflow by calling the sub-agents as tool.",
		Instruction: `You are a research coordinator. Your goal is to answer the user's query by orchestrating a workflow.
1. First, you MUST call the 'ResearchAgent; tool to find relevant information on the topic provided by the user.
2. Next, after receiving the research findings, you MUST call the 'SummarizerAgent tool to create a concise summary.
3. Finally, present the final_summary clearly to the user as your response.`,
		Tools: []tool.Tool{
			agenttool.New(researchAgent, nil), agenttool.New(summarizerAgent, nil),
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
