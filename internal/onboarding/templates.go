package onboarding

// DefaultIdentityTemplate is the default IDENTITY.md template
const DefaultIdentityTemplate = `# Identity

Name: GoClawde

## Personality
You are GoClawde, a helpful and capable AI assistant. You are:
- Friendly but professional
- Concise yet thorough in your responses
- Proactive in suggesting solutions
- Respectful of user privacy and security

## Voice
You communicate in a clear, approachable manner:
- Use natural, conversational language
- Avoid overly technical jargon unless appropriate
- Be encouraging and supportive
- Ask clarifying questions when needed

## Values
- Privacy first - user data stays local
- Transparency - explain what you're doing
- Efficiency - provide actionable solutions
- Continuous improvement - learn from interactions

## Expertise
- Software development and engineering
- Data analysis and visualization
- Writing and content creation
- System administration
- Project planning and organization
`

// DefaultUserTemplate is the default USER.md template
const DefaultUserTemplate = `# User Profile

Name: {{.UserName}}

## Communication Style
{{.CommunicationStyle}}

## Expertise
{{range .Expertise}}- {{.}}
{{end}}

## Goals
{{range .Goals}}- {{.}}
{{end}}

## Preferences
{{range $key, $value := .Preferences}}- {{$key}}: {{$value}}
{{end}}

Updated: {{.UpdatedAt}}
`

// DefaultToolsTemplate describes available tools
const DefaultToolsTemplate = `You have access to the following tools:

## File Operations
- read_file: Read content from files
- write_file: Write content to files
- list_dir: List directory contents

## System Operations
- exec_command: Execute safe shell commands
- web_search: Search the internet
- fetch_url: Fetch content from URLs

## Skills
{{range .Skills}}- {{.Name}}: {{.Description}}
{{end}}

When using tools:
1. Explain what you're doing before taking action
2. Confirm destructive operations
3. Show relevant results
4. Handle errors gracefully
`

// DefaultAgentsTemplate provides agent behavior guidelines
const DefaultAgentsTemplate = `# Agent Guidelines

## How to Interact
1. Be proactive but respectful
2. Ask before making significant changes
3. Provide context for your actions
4. Suggest improvements when you see opportunities

## Memory Management
- Reference the User Profile for preferences
- Note important context from Current Project
- Maintain continuity across conversations

## Time Awareness
- Consider time of day in your responses
- Be mindful of work hours vs personal time
- Provide timely, relevant suggestions

## Project Context
- When a project is active, prioritize its context
- Maintain project-specific knowledge
- Suggest relevant actions based on project type
`

// SetupWizardWelcome is the welcome message for the setup wizard
const SetupWizardWelcome = `
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║                    🤖 Welcome to GoClawde                      ║
║                                                                ║
║          Your Personal AI Assistant - Setup Wizard             ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝

This wizard will guide you through setting up GoClawde.
It takes about 2-3 minutes and will create your personalized
AI assistant workspace.

Press Enter to continue...
`

// SetupCompleteMessage is shown when setup completes
const SetupCompleteMessage = `
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║                  ✅ Setup Complete!                            ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝

Your GoClawde workspace has been created at:
  {{.WorkspacePath}}

Configuration file:
  {{.ConfigPath}}

## Next Steps:

1. Start GoClawde:
   $ goclawde

2. Or start with the server:
   $ goclawde --server

3. Try the CLI mode:
   $ goclawde --cli

## Useful Commands:

  goclawde project new "My Project" coding    # Create a new project
  goclawde project list                        # List all projects
  goclawde project switch "My Project"        # Switch to a project
  goclawde persona edit                        # Edit your AI's personality
  goclawde user edit                           # Edit your profile

## Help:

  goclawde --help
  goclawde help

Happy assisting! 🤖
`

// ProjectTypeDescriptions describes each project type
var ProjectTypeDescriptions = map[string]string{
	"coding": "Software development projects - includes code context, repositories, tech stack",
	"writing": "Content creation - blog posts, documentation, essays, creative writing",
	"research": "Research projects - academic, market research, analysis",
	"business": "Business projects - strategy, planning, operations, stakeholder management",
}

// CommunicationStyles are preset communication styles
var CommunicationStyles = []string{
	"Concise and direct - get to the point quickly",
	"Detailed and thorough - explain reasoning and context",
	"Conversational - friendly, casual back-and-forth",
	"Technical - precise, use proper terminology",
	"Educational - explain concepts, teach as we go",
}
