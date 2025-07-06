export function tooltip(node) {
	function updateTooltipPosition(event) {
		const rect = node.getBoundingClientRect();

		// Calculate position
		const tooltipX = rect.left + rect.width * 2.3;
		const tooltipY = rect.top * 1.01;

		node.style.setProperty('--tooltip-x', `${tooltipX}px`);
		node.style.setProperty('--tooltip-y', `${tooltipY}px`);
	}

	node.addEventListener('mouseenter', updateTooltipPosition);

	return {
		destroy() {
			node.removeEventListener('mouseenter', updateTooltipPosition);
		}
	};
}
