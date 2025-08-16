// Theme handling
(function(){
    function applyThemeClass() {
        if (localStorage.theme === 'dark' || (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }

    function initTheme() {
        applyThemeClass();
    }

    function toggleTheme() {
        if (document.documentElement.classList.contains('dark')) {
            document.documentElement.classList.remove('dark');
            localStorage.theme = 'light';
        } else {
            document.documentElement.classList.add('dark');
            localStorage.theme = 'dark';
        }
    }

    // Expose globally for onclick handlers
    window.toggleTheme = toggleTheme;

    // Initial run as early as possible
    applyThemeClass();

    // Run on traditional page load
    document.addEventListener('DOMContentLoaded', initTheme);

    // Run after htmx swaps new content into the DOM
    document.addEventListener('htmx:load', initTheme);

    // Handle bfcache restores (back/forward navigation)
    window.addEventListener('pageshow', initTheme);
})();