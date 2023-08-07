# GoPlay
Generic game matchmaker.
* Supports N vs N vs ... match format: 1 vs 1 (e.g. fighting), 5 vs 5 (e.g. MOBA), 3 vs 3 vs 3 vs ... (e.g. battle royale)
* Checks if players ready for match before starting a server
* Set rating range to search for players with approximately the same skill
* Configured in matchmaker_config.json

# Interaction with other services
* Player data - get player info like rating, winrate, ping, etc.
* Server manager - request new game server instance
* Lobby (optional) - for group search (more than 1 vs 1 player)

![Interaction with lobby](/docs/lobby_interaction.jpg)
