package main

import "charm.land/lipgloss/v2"

const asciiLogo = ` 
███████╗ ██████╗ ██████╗ ████████╗██████╗ ██╗   ██╗████████╗███████╗
██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝
█████╗  ██║   ██║██████╔╝   ██║   ██████╔╝ ╚████╔╝    ██║   █████╗
██╔══╝  ██║   ██║██╔══██╗   ██║   ██╔══██╗  ╚██╔╝     ██║   ██╔══╝
██║     ╚██████╔╝██║  ██║   ██║   ██████╔╝   ██║      ██║   ███████╗
╚═╝      ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═════╝    ╚═╝      ╚═╝   ╚══════╝

            Defend Your Secrets. Secure Every Byte.

                                  |>>>
                                  |
                    |>>>      _  _|_  _         |>>>
                    |        |;| |;| |;|        |
                _  _|_  _    \\.    .  /    _  _|_  _
               |;|_|;|_|;|    \\:. ,  /    |;|_|;|_|;|
               \\..      /    ||;   . |    \\.    .  /
                \\.  ,  /     ||:  .  |     \\:  .  /
                 ||:   |_   _ ||_ . _ | _   _||:   |
                 ||:  .|||_|;|_|;|_|;|_|;|_|;||:.  |
                 ||:   ||.    .     .      . ||:  .|
                 ||: . || .     . .   .  ,   ||:   |   
                 ||:   ||:  ,  _______   .   ||: , |        
                 ||:   || .   /+++++++\    . ||:   |
                 ||:   ||.    |+++++++| .    ||: . |
                 ||: . ||: ,  |+++++++|.  . _||_   |         
				--------------\+++++++/--------------
`

// ponytail: minimal theme — one style per use, no abstraction layers.
var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginBottom(1)

	styleProject = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")).
			Bold(true)

	styleEnv = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117"))

	styleSecret = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	styleUID = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	styleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true)

	styleWarning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	styleLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)
