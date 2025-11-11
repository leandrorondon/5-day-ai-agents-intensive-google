package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/parallelagent"
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

	techResearcher, err := llmagent.New(llmagent.Config{
		Name:        "TechResearcher",
		Model:       model,
		Description: "Focuses on AI and ML trends.",
		Instruction: `Research the latest AI/ML trends. Include 3 key developments,
	the main companies involved, and the potential impact. Keep the report very concise (100 words).`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "tech_research",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	healthResearcher, err := llmagent.New(llmagent.Config{
		Name:        "healthResearcher",
		Model:       model,
		Description: "Focuses on medical breakthroughs.",
		Instruction: `Research recent medical breakthroughs. Include 3 significant advances,
their practical applications, and estimated timelines. Keep the report concise (100 words).`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "health_research",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	financeResearch, err := llmagent.New(llmagent.Config{
		Name:        "FinanceResearcher",
		Model:       model,
		Description: "Focuses on fintech trends.",
		Instruction: `Research current fintech trends. Include 3 key trends,
their market implications, and the future outlook. Keep the report concise (100 words).`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		OutputKey: "finance_research",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	aggregatorAgent, err := llmagent.New(llmagent.Config{
		Name:        "AggregatorAgent",
		Model:       model,
		Description: "The AggregatorAgent runs *after* the parallel step to synthesize the results.",
		Instruction: `Combine these three research findings into a single executive summary:

    **Technology Trends:**
    {tech_research}
    
    **Health Breakthroughs:**
    {health_research}
    
    **Finance Innovations:**
    {finance_research}
    
    Your summary should highlight common themes, surprising connections, and the most important key takeaways from all three reports. The final summary should be around 200 words.`,
		OutputKey: "executive_summary",
	})

	parallelResearchTeam, err := parallelagent.New(parallelagent.Config{
		AgentConfig: agent.Config{
			Name:        "ParallelResearchTeam",
			Description: "Runs all its sub-agents simultaneously.",
			SubAgents:   []agent.Agent{techResearcher, healthResearcher, financeResearch},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	rootAgent, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "RootAgent",
			Description: "Defines the high-level workflow: run the parallel team first, then run the aggregator.",
			SubAgents:   []agent.Agent{parallelResearchTeam, aggregatorAgent},
		},
	})

	config := &adk.Config{
		AgentLoader: services.NewSingleAgentLoader(rootAgent),
	}

	l := full.NewLauncher()
	err = l.Execute(ctx, config, os.Args[1:])
	if err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
