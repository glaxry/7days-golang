(() => {
  const root = document.documentElement;
  const themeButton = document.querySelector('.theme-button');
  const savedTheme = localStorage.getItem('7days-theme');
  const preferredTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';

  const setTheme = (theme) => {
    root.dataset.theme = theme;
    if (themeButton) {
      themeButton.setAttribute('aria-label', theme === 'dark' ? '切换到浅色主题' : '切换到深色主题');
    }
  };

  setTheme(savedTheme || preferredTheme);
  themeButton?.addEventListener('click', () => {
    const next = root.dataset.theme === 'dark' ? 'light' : 'dark';
    localStorage.setItem('7days-theme', next);
    setTheme(next);
  });

  const menuButton = document.querySelector('.menu-button');
  const overlay = document.querySelector('.sidebar-overlay');
  const closeMenu = () => {
    document.body.classList.remove('nav-open');
    menuButton?.setAttribute('aria-expanded', 'false');
  };
  menuButton?.addEventListener('click', () => {
    const open = document.body.classList.toggle('nav-open');
    menuButton.setAttribute('aria-expanded', String(open));
  });
  overlay?.addEventListener('click', closeMenu);
  document.querySelectorAll('.sidebar a').forEach((link) => link.addEventListener('click', closeMenu));
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') closeMenu();
  });

  const search = document.querySelector('#nav-search');
  const navItems = [...document.querySelectorAll('.nav-item')];
  const navGroups = [...document.querySelectorAll('.nav-group')];
  const emptyState = document.querySelector('.search-empty');
  search?.addEventListener('input', () => {
    const query = search.value.trim().toLocaleLowerCase('zh-CN');
    let visibleCount = 0;
    navItems.forEach((item) => {
      const visible = !query || item.dataset.search.includes(query);
      item.hidden = !visible;
      if (visible) visibleCount += 1;
    });
    navGroups.forEach((group) => {
      group.hidden = ![...group.querySelectorAll('.nav-item')].some((item) => !item.hidden);
    });
    if (emptyState) emptyState.hidden = visibleCount !== 0;
  });

  document.querySelectorAll('.prose pre').forEach((pre) => {
    const button = document.createElement('button');
    button.className = 'copy-button';
    button.type = 'button';
    button.textContent = '复制';
    button.setAttribute('aria-label', '复制代码');
    button.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(pre.innerText);
        button.textContent = '已复制';
        setTimeout(() => { button.textContent = '复制'; }, 1400);
      } catch {
        button.textContent = '复制失败';
      }
    });
    pre.append(button);
  });

  const progress = document.querySelector('.reading-progress span');
  const updateProgress = () => {
    if (!progress) return;
    const scrollable = document.documentElement.scrollHeight - window.innerHeight;
    const value = scrollable > 0 ? Math.min(1, window.scrollY / scrollable) : 0;
    progress.style.transform = `scaleX(${value})`;
  };
  updateProgress();
  document.addEventListener('scroll', updateProgress, { passive: true });

  const tocLinks = [...document.querySelectorAll('.toc a')];
  const headings = tocLinks
    .map((link) => document.getElementById(decodeURIComponent(link.hash.slice(1))))
    .filter(Boolean);
  if (headings.length) {
    const observer = new IntersectionObserver((entries) => {
      const visible = entries.filter((entry) => entry.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
      if (!visible.length) return;
      tocLinks.forEach((link) => link.classList.toggle('is-active', link.hash.slice(1) === visible[0].target.id));
    }, { rootMargin: '-15% 0px -70% 0px' });
    headings.forEach((heading) => observer.observe(heading));
  }
})();
