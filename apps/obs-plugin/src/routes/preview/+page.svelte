<script lang="ts">
	import { onMount } from 'svelte';
	import { mockItems, previewPinnedSCs } from '$lib/mockData.js';
	import ChatList from '$lib/ChatList.svelte';
	import SuperChatPin from '$lib/SuperChatPin.svelte';

	let theme = $state('plain');

	onMount(() => {
		theme = new URLSearchParams(window.location.search).get('theme') ?? 'plain';
	});
</script>

<div class="preview-wrap">
	<!-- Neutral checkerboard to show transparency, like design tools -->
	<div class="checker" aria-hidden="true"></div>
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
		background-position: 0 0, 0 10px, 10px -10px, -10px 0px;
	}

	.overlay {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		background: transparent;
		overflow: hidden;
	}
</style>
