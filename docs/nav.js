// Willow Documentation  -  Navigation Data & Sidebar Logic

const REFERENCE_ITEMS = [
    { label: "API Reference", href: "https://pkg.go.dev/github.com/phanxgames/willow" },
    { label: "GitHub", href: "https://github.com/phanxgames/willow" },
    { label: "Donate", page: "donate" },
];

const NAV_TABS = [
    { id: "docs", label: "Docs", sections: [
        {
            title: "Getting Started",
            items: [
                { label: "What is Willow?", page: "what-is-willow" },
                { label: "Getting Started", page: "getting-started" },
                { label: "Architecture", page: "architecture" },
                { label: "Performance", page: "performance-overview" },
            ]
        },
        {
            title: "Core",
            items: [
                { label: "Scene", page: "scene" },
                { label: "Nodes", page: "nodes" },
                { label: "Node Index", page: "node-index" },
                { label: "Transforms", page: "transforms" },
            ]
        },
        {
            title: "Rendering",
            items: [
                { label: "Solid-Color Sprites", page: "solid-color-sprites" },
                { label: "Sprites", page: "sprites" },
                { label: "Atlases", page: "atlases" },
                { label: "Dynamic Atlas", page: "dynamic-atlas" },
                { label: "Camera & Viewport", page: "camera-and-viewport" },
                { label: "Text & Fonts", page: "text-and-fonts" },
                { label: "Tilemap Viewport", page: "tilemap-viewport" },
                { label: "Polygons", page: "polygons" },
                { label: "Offscreen Rendering", page: "offscreen-rendering" },
            ]
        },
        {
            title: "Input",
            items: [
                { label: "Input & Hit Testing", page: "input-hit-testing-and-gestures" },
                { label: "Events & Callbacks", page: "events-and-callbacks" },
            ]
        },
        {
            title: "Effects",
            items: [
                { label: "Tweens & Animation", page: "tweens-and-animation" },
                { label: "Particles", page: "particles" },
                { label: "Mesh & Distortion", page: "meshes" },
                { label: "Ropes", page: "ropes" },
                { label: "Lighting", page: "lighting" },
                { label: "Clipping & Masks", page: "clipping-and-masks" },
                { label: "Post-Processing Filters", page: "post-processing-filters" },
            ]
        },
        {
            title: "Caching",
            items: [
                { label: "CacheAsTree", page: "cache-as-tree" },
                { label: "CacheAsTexture", page: "cache-as-texture" },
            ]
        },
        {
            title: "Tools",
            items: [
                { label: "Debug & Testing", page: "debug-and-testing" },
                { label: "ECS Integration", page: "ecs-integration" },
                { label: "Cmd: atlaspack", page: "atlaspack" },
                { label: "Cmd: fontgen", page: "fontgen" },
                { label: "LLM Guide", page: "llm" },
            ]
        },
    ]},
    { id: "examples", label: "Examples", sections: [
        { title: "Intro", items: [
            { label: "Intro", page: "examples" },
        ]},
        { title: "WASM Demos", items: [
            { label: "10k Sprites", page: "examples", anchor: "10k-sprites" },
            { label: "Physics Shapes", page: "examples", anchor: "physics-shapes" },
            { label: "Rope Garden", page: "examples", anchor: "rope-garden" },
            { label: "Tween Gallery", page: "examples", anchor: "tween-gallery" },
            { label: "Filter Gallery", page: "examples", anchor: "filter-gallery" },
            { label: "Lighting Demo", page: "examples", anchor: "lighting-demo" },
            { label: "Underwater", page: "examples", anchor: "underwater" },
        ]},
        { title: "Basics", items: [
            { label: "Basic", page: "examples", anchor: "basic" },
            { label: "Shapes", page: "examples", anchor: "shapes" },
            { label: "Interaction", page: "examples", anchor: "interaction" },
        ]},
        { title: "Text", items: [
            { label: "SpriteFont", page: "examples", anchor: "spritefont" },
            { label: "SpriteFont (TTF)", page: "examples", anchor: "spritefont-ttf" },
            { label: "PixelFont", page: "examples", anchor: "pixelfont" },
        ]},
        { title: "Animation", items: [
            { label: "Tweens", page: "examples", anchor: "tweens" },
            { label: "Tween Gallery", page: "examples", anchor: "tween-gallery-ex" },
            { label: "Particles", page: "examples", anchor: "particles" },
        ]},
        { title: "Visual Effects", items: [
            { label: "Shaders", page: "examples", anchor: "shaders" },
            { label: "Filter Gallery", page: "examples", anchor: "filter-gallery-ex" },
            { label: "Outline", page: "examples", anchor: "outline" },
            { label: "Masks", page: "examples", anchor: "masks" },
            { label: "Lighting", page: "examples", anchor: "lighting" },
        ]},
        { title: "Sprites & Maps", items: [
            { label: "Atlas", page: "examples", anchor: "atlas" },
            { label: "Tilemap", page: "examples", anchor: "tilemap" },
            { label: "Tilemap Viewport", page: "examples", anchor: "tilemap-viewport" },
        ]},
        { title: "Meshes", items: [
            { label: "Rope", page: "examples", anchor: "rope" },
            { label: "Water Mesh", page: "examples", anchor: "water-mesh" },
        ]},
        { title: "Showcases", items: [
            { label: "10k Sprites", page: "examples", anchor: "sprites10k" },
            { label: "Physics Shapes", page: "examples", anchor: "physics" },
            { label: "Rope Garden", page: "examples", anchor: "ropegarden" },
            { label: "Underwater", page: "examples", anchor: "underwater-ex" },
        ]},
    ]},
    { id: "tutorials", label: "Tutorials", sections: [
        { title: "Tutorials", items: [{ label: "Tutorials (Coming Soon)", page: "tutorials" }] }
    ]},
    // { id: "reference", label: "Reference", sections: [
    //     { title: "Reference", items: REFERENCE_ITEMS }
    // ]}
];

let activeTabId = "docs";

function getActivePage() {
    const params = new URLSearchParams(window.location.search);
    return params.get("page") || "what-is-willow";
}

function findTabForPage(page) {
    for (const tab of NAV_TABS) {
        for (const section of tab.sections) {
            for (const item of section.items) {
                if (item.page === page) return tab.id;
            }
        }
    }
    return "docs";
}

function buildTabs() {
    const container = document.getElementById("sidebar-tabs");
    container.innerHTML = "";

    NAV_TABS.forEach(tab => {
        const btn = document.createElement("button");
        btn.className = "sidebar-tab" + (tab.id === activeTabId ? " active" : "");
        btn.textContent = tab.label;
        btn.addEventListener("click", () => switchTab(tab.id, true));
        container.appendChild(btn);
    });
}

function switchTab(tabId, autoNavigate) {
    activeTabId = tabId;

    document.querySelectorAll(".sidebar-tab").forEach(btn => {
        btn.classList.toggle("active", btn.textContent === NAV_TABS.find(t => t.id === tabId).label);
    });

    const tab = NAV_TABS.find(t => t.id === tabId);
    if (tab) buildSidebar(tab.sections);

    // Navigate to the first page in the tab when clicked directly
    if (autoNavigate && tab) {
        for (const section of tab.sections) {
            for (const item of section.items) {
                if (item.page) {
                    navigateTo(item.page, item.anchor);
                    return;
                }
            }
        }
    }
}

function buildFooter() {
    const footer = document.getElementById("sidebar-footer");
    footer.innerHTML = "";

    REFERENCE_ITEMS.forEach(item => {
        const a = document.createElement("a");
        a.className = "sidebar-footer-link";
        a.textContent = item.label;
        if (item.href) {
            a.href = item.href;
            a.target = "_blank";
            a.rel = "noopener noreferrer";
        } else if (item.page) {
            a.href = `?page=${item.page}`;
            a.addEventListener("click", (e) => {
                e.preventDefault();
                navigateTo(item.page);
            });
        }
        footer.appendChild(a);
    });
}

function buildSidebar(sections) {
    const nav = document.getElementById("sidebar-nav");
    const activePage = getActivePage();
    const activeAnchor = window.location.hash ? window.location.hash.slice(1) : null;
    nav.innerHTML = "";

    sections.forEach(section => {
        const sectionEl = document.createElement("div");
        sectionEl.className = "nav-section";

        const titleEl = document.createElement("div");
        titleEl.className = "nav-section-title";
        titleEl.innerHTML = `${section.title} <span class="chevron">&#9662;</span>`;
        titleEl.addEventListener("click", () => {
            sectionEl.classList.toggle("collapsed");
        });
        sectionEl.appendChild(titleEl);

        const listEl = document.createElement("ul");
        listEl.className = "nav-items";

        section.items.forEach(item => {
            const li = document.createElement("li");
            li.className = "nav-item";
            const a = document.createElement("a");
            a.textContent = item.label;
            if (item.href) {
                a.href = item.href;
                a.target = "_blank";
                a.rel = "noopener noreferrer";
            } else if (item.anchor) {
                a.href = `?page=${item.page}#${item.anchor}`;
                if (item.page === activePage && item.anchor === activeAnchor) {
                    a.className = "active";
                }
                a.addEventListener("click", (e) => {
                    e.preventDefault();
                    navigateTo(item.page, item.anchor);
                });
            } else {
                a.href = `?page=${item.page}`;
                if (item.page === activePage) {
                    a.className = "active";
                }
                a.addEventListener("click", (e) => {
                    e.preventDefault();
                    navigateTo(item.page);
                });
            }
            li.appendChild(a);
            listEl.appendChild(li);
        });

        sectionEl.appendChild(listEl);
        nav.appendChild(sectionEl);
    });
}

function scrollToActive() {
    const activeLink = document.querySelector(".nav-item a.active");
    if (activeLink) {
        activeLink.scrollIntoView({ block: "center", behavior: "instant" });
    }
}

function navigateTo(page, anchor) {
    if (!page) return;
    const url = anchor ? `?page=${page}#${anchor}` : `?page=${page}`;
    history.pushState({page, anchor}, "", url);

    // Switch to correct tab if needed
    const tabId = findTabForPage(page);
    if (tabId !== activeTabId) {
        switchTab(tabId);
    }

    // Update active state in sidebar
    document.querySelectorAll(".nav-item a").forEach(a => {
        const href = a.getAttribute("href");
        const isActive = anchor
            ? href === `?page=${page}#${anchor}`
            : href === `?page=${page}`;
        a.classList.toggle("active", isActive);
    });

    scrollToActive();

    // Load page in iframe
    const iframe = document.getElementById("content-frame");
    const iframeSrc = anchor
        ? `viewer.html?page=${page}#${anchor}`
        : `viewer.html?page=${page}`;

    // If only the anchor changed (same page), setting iframe.src won't
    // reload  -  just scroll within the already-loaded document instead.
    const currentPage = new URL(iframe.src, location.href).searchParams.get("page");
    if (currentPage === page && anchor) {
        try {
            const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
            const target = iframeDoc.getElementById(anchor);
            if (target) {
                target.scrollIntoView({ behavior: "instant" });
            } else {
                iframe.src = iframeSrc;
            }
        } catch (e) {
            iframe.src = iframeSrc;
        }
    } else {
        iframe.src = iframeSrc;
    }

    // Close mobile sidebar
    document.querySelector(".sidebar").classList.remove("open");
    document.querySelector(".overlay").classList.remove("visible");
}

window.addEventListener("popstate", (e) => {
    const page = (e.state && e.state.page) || getActivePage();
    const anchor = e.state && e.state.anchor;

    const tabId = findTabForPage(page);
    if (tabId !== activeTabId) {
        switchTab(tabId);
    }

    const iframe = document.getElementById("content-frame");
    const iframeSrc = anchor
        ? `viewer.html?page=${page}#${anchor}`
        : `viewer.html?page=${page}`;

    // Same-page anchor change: scroll within the already-loaded iframe.
    const currentPage = new URL(iframe.src, location.href).searchParams.get("page");
    if (currentPage === page && anchor) {
        try {
            const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
            const target = iframeDoc.getElementById(anchor);
            if (target) {
                target.scrollIntoView({ behavior: "instant" });
            } else {
                iframe.src = iframeSrc;
            }
        } catch (e) {
            iframe.src = iframeSrc;
        }
    } else {
        iframe.src = iframeSrc;
    }

    document.querySelectorAll(".nav-item a").forEach(a => {
        const href = a.getAttribute("href");
        const url = anchor ? `?page=${page}#${anchor}` : `?page=${page}`;
        a.classList.toggle("active", href === url);
    });

    scrollToActive();
});

document.addEventListener("DOMContentLoaded", () => {
    const activePage = getActivePage();
    activeTabId = findTabForPage(activePage);

    buildTabs();
    buildFooter();
    switchTab(activeTabId);

    // Hamburger toggle
    const hamburger = document.querySelector(".hamburger");
    const sidebar = document.querySelector(".sidebar");
    const overlay = document.querySelector(".overlay");

    if (hamburger) {
        hamburger.addEventListener("click", () => {
            sidebar.classList.toggle("open");
            overlay.classList.toggle("visible");
        });
    }

    if (overlay) {
        overlay.addEventListener("click", () => {
            sidebar.classList.remove("open");
            overlay.classList.remove("visible");
        });
    }

    scrollToActive();

    // Load initial page in iframe
    const iframe = document.getElementById("content-frame");
    iframe.src = `viewer.html?page=${activePage}`;
});
