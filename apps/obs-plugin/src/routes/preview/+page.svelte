<script lang="ts">
	import { onMount } from 'svelte';
	import { mockItems, previewPinnedSCs } from '$lib/mockData.js';
	import ChatList from '$lib/ChatList.svelte';
	import SuperChatPin from '$lib/SuperChatPin.svelte';

	let theme = $state('plain');

	const KNOWN_THEMES = ['plain', 'bubble', 'bubble-light'];

	onMount(() => {
		const t = new URLSearchParams(window.location.search).get('theme') || '';
		theme = KNOWN_THEMES.includes(t) ? t : 'plain';
	});

	const isLight = $derived(theme === 'bubble-light');
</script>

<div class="preview-wrap">
	<!-- Neutral checkerboard to show transparency, like design tools -->
	<div class="checker" class:light={isLight} aria-hidden="true"></div>
	<div class="overlay theme-{theme}">
		<SuperChatPin superchats={previewPinnedSCs} />
		<ChatList items={mockItems} />
	</div>
</div>

<style>
	.preview-wrap {
		position: relative;
		width: 100vw;
		height: 100vh;
		overflow: hidden;
	}

	/* Neutral checkerboard — shows transparent areas clearly */
	.checker {
		position: absolute;
		inset: 0;
		background-color: #888;
		background-image:
			linear-gradient(45deg, #aaa 25%, transparent 25%),
			linear-gradient(-45deg, #aaa 25%, transparent 25%),
			linear-gradient(45deg, transparent 75%, #aaa 75%),
			linear-gradient(-45deg, transparent 75%, #aaa 75%);
		background-size: 20px 20px;
		background-position:
			0 0,
			0 10px,
			10px -10px,
			-10px 0px;
		transition: background 0.3s;
	}

	.checker.light {
		background-color: #d0d0d0;
		background-image:
			linear-gradient(45deg, #e8e8e8 25%, transparent 25%),
			linear-gradient(-45deg, #e8e8e8 25%, transparent 25%),
			linear-gradient(45deg, transparent 75%, #e8e8e8 75%),
			linear-gradient(-45deg, transparent 75%, #e8e8e8 75%);
	}

	.overlay {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		max-width: var(--max-width, none);
		background: transparent;
		overflow: hidden;
	}
</style>
