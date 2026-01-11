.PHONY: run glm minimax

run: minimax

minimax:
	ANTHROPIC_API_KEY=$$(grep AUTH ~/.minimax | cut -d= -f2) \
	ANTHROPIC_BASE_URL=https://api.minimax.io/anthropic \
	ANTHROPIC_MODEL=MiniMax-M2.1 \
	go run .

glm:
	ANTHROPIC_API_KEY=$$(grep AUTH ~/.glm | cut -d= -f2) \
	ANTHROPIC_BASE_URL=https://api.z.ai/api/anthropic \
	ANTHROPIC_MODEL=glm-4.7 \
	go run .
