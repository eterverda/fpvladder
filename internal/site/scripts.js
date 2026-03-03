document.addEventListener('click', (e) => {
    const target = e.target.closest('[data-href]');
    if (target && !e.target.closest('a')) {
        window.location.href = target.dataset.href;
    }
});
