/**
 * Action to detect clicks outside of an element
 * Usage: <div use:clickOutside={{ enabled: boolean, cb: () => void }}>
 */
export function clickOutside(node, { enabled = true, cb = () => {} }) {
	const handleClick = (event) => {
		if (!enabled) return;
		if (node && !node.contains(event.target) && !event.defaultPrevented) {
			cb();
		}
	};

	document.addEventListener('click', handleClick, true);

	return {
		update(params) {
			// Update enabled state and callback if they change
			if (params) {
				enabled = params.enabled;
				if (params.cb) cb = params.cb;
			}
		},
		destroy() {
			document.removeEventListener('click', handleClick, true);
		}
	};
}
