package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/loopagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/server/restapi/services"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/exitlooptool"
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

	initialWriterAgent, err := llmagent.New(llmagent.Config{
		Name:        "InitialWriterAgent",
		Model:       model,
		Description: "This agent runs ONCE at the beginning to create the first draft",
		Instruction: `Based on the user's prompt, write the first draft of a short story (around 100-150 words).
	Output only the story text, with no introduction or explanation.`,
		OutputKey: "current_story",
	})

	criticAgent, err := llmagent.New(llmagent.Config{
		Name:        "CriticAgent",
		Model:       model,
		Description: "This agent's only job is to provide feedback or the approval signal. It has no tools.",
		Instruction: `You are a constructive story critic. Review the story provided below.
    Story: {current_story}
    
    Evaluate the story's plot, characters, and pacing.
    - If the story is well-written and complete, you MUST respond with the exact phrase: "APPROVED"
    - Otherwise, provide 2-3 specific, actionable suggestions for improvement.`,
		OutputKey: "critique",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	exitLoop, err := exitlooptool.New()
	if err != nil {
		log.Fatalf("Failed to create exitLoop: %v", err)
	}

	refinerAgent, err := llmagent.New(llmagent.Config{
		Name:        "RefinerAgent",
		Model:       model,
		Description: "This agent refines the story based on critique OR calls the exit_loop function",
		Instruction: `You are a story refiner. You have a story draft and critique.
    
			Story Draft: {current_story}
			Critique: {critique}
			
			Your task is to analyze the critique.
			- IF the critique is EXACTLY "APPROVED", you MUST call the 'exit_loop' function and nothing else.
			- OTHERWISE, rewrite the story draft to fully incorporate the feedback from the critique.`,
		OutputKey: "current_story",
		Tools: []tool.Tool{
			exitLoop,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	storyRefinementLoop, err := loopagent.New(loopagent.Config{
		AgentConfig: agent.Config{
			Name:        "StoryRefinementLoop",
			Description: "",
			SubAgents:   []agent.Agent{criticAgent, refinerAgent},
		},
		MaxIterations: 2,
	})

	rootAgent, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "StoryPipeline",
			Description: "Root Agent for the story pipeline.",
			SubAgents:   []agent.Agent{initialWriterAgent, storyRefinementLoop},
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
