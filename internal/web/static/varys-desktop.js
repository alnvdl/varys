let desktop_elements;

function desktop_init() {
    desktop_elements = {
        feedList: document.querySelector("#desktop-feed-list"),
        feedTitle: document.querySelector("#desktop-feed-title"),
        feedReadButton: document.querySelector("#desktop-feed-read-button"),
        allReadButton: document.querySelector("#desktop-all-read-button"),
        itemList: document.querySelector("#desktop-item-list"),
        itemTitle: document.querySelector("#desktop-item-title"),
        itemContent: document.querySelector("#desktop-item-content"),
        openButton: document.querySelector("#desktop-open-button"),
    };
}

function desktop_set_content(element, ...content) {
    element.replaceChildren(...content);
}

function desktop_spinner() {
    return create_element("div", {
        class_name: "desktop-spinner",
        children: [create_element("div", {class_name: "spinner"})],
    });
}

function desktop_set_panel_loading(...elements) {
    elements.forEach(element => desktop_set_content(element, desktop_spinner()));
}

function desktop_set_loading() {
    desktop_elements.feedTitle.textContent = "...";
    desktop_elements.itemTitle.textContent = "...";
    desktop_set_read_button(desktop_elements.allReadButton);
    desktop_set_read_button(desktop_elements.feedReadButton);
    desktop_set_panel_loading(
        desktop_elements.feedList,
        desktop_elements.itemList,
        desktop_elements.itemContent,
    );
}

function desktop_set_item_list_loading() {
    desktop_set_read_button(desktop_elements.feedReadButton);
    desktop_set_panel_loading(desktop_elements.itemList);
}

function desktop_set_feed_list_loading() {
    desktop_set_read_button(desktop_elements.allReadButton);
    desktop_set_panel_loading(desktop_elements.feedList);
}

function desktop_set_feed_view_loading() {
    desktop_elements.feedTitle.textContent = "...";
    desktop_set_read_button(desktop_elements.feedReadButton);
    desktop_set_item_list_loading();
}

function desktop_set_item_view_loading() {
    desktop_set_item_content();
    desktop_elements.itemTitle.textContent = "...";
    desktop_set_panel_loading(desktop_elements.itemContent);
}

function desktop_set_error(message) {
    console.trace(message);
    let error = create_element("div", {class_name: "error", text: message});
    desktop_set_content(desktop_elements.feedList, error.cloneNode(true));
    desktop_set_content(desktop_elements.itemList, error.cloneNode(true));
    desktop_set_content(desktop_elements.itemContent, error);
}

function desktop_set_open_button(item) {
    let open_button = desktop_elements.openButton;
    open_button.hidden = !item;
    if (!item) {
        open_button.removeAttribute("href");
        open_button.removeAttribute("target");
        open_button.removeAttribute("rel");
        return;
    }

    open_button.setAttribute("href", item.url);
    open_button.setAttribute("target", "_blank");
    open_button.setAttribute("rel", "noopener noreferrer");
}

function desktop_set_item_content(item) {
    let title = desktop_elements.itemTitle;
    title.replaceChildren();
    if (item) {
        title.append(gen_item_header(item, {list_view: true}));
    }
    desktop_set_open_button(item);

    if (item) {
        desktop_set_content(desktop_elements.itemContent, gen_item_content(item));
    } else {
        desktop_set_content(desktop_elements.itemContent, create_element("div", {
            class_name: "empty-message",
            text: "No item selected.",
        }));
    }
}

function desktop_set_read_button(
    button,
    feed,
    {
        loading = desktop_set_item_list_loading,
        refresh = {loading: "items"},
    } = {},
) {
    button.onclick = null;
    button.hidden = !feed;
    if (!feed) return;

    button.onclick = async () => {
        loading();
        await mark_feed_as_read(feed.uid, feed.last_updated);
        await desktop_refresh(refresh);
    };
}

function desktop_update_read_state(feeds, feed, item) {
    if (!item.read) return;

    let feed_item = feed.items?.find(feed_item => feed_item.uid === item.uid);
    if (feed_item && !feed_item.read) {
        feed_item.read = true;
    }

    feeds.forEach(feed_summary => {
        if (feed_summary.uid === "all" || feed_summary.uid === item.feed_uid) {
            feed_summary.read_count++;
        }
    });
}

function desktop_render_feed_list(feeds, selected_uid) {
    desktop_feeds = feeds;
    desktop_set_content(desktop_elements.feedList, gen_feed_list(feeds, {
        selected_uid,
        link_handler: desktop_select_feed,
    }));

    let all_feed = feeds.find(feed => feed.uid === "all");
    desktop_set_read_button(desktop_elements.allReadButton, all_feed,
        {loading: desktop_set_feed_list_loading, refresh: {loading: false}},
    );
}

function desktop_set_feed_selection(selected_uid) {
    desktop_elements.feedList.querySelectorAll("[data-feed-uid]").forEach(feed_item => {
        feed_item.classList.toggle("feed-selected", feed_item.dataset.feedUid === selected_uid);
    });
}

function desktop_set_item_selection(selected_uid) {
    desktop_elements.itemList.querySelectorAll("[data-item-uid]").forEach(item_link => {
        item_link.classList.toggle("item-link-selected", item_link.dataset.itemUid === selected_uid);
    });
}

function desktop_select_feed() {
    let state = get_current_state();
    let selected_uid = state.parts[1] || "all";
    desktop_set_feed_selection(selected_uid);
    desktop_set_item_selection("");
    desktop_set_feed_view_loading();
    desktop_set_item_content();
    desktop_refresh({loading: "items", feed_switch: true});
}

function desktop_select_item() {
    let state = get_current_state();
    let selected_uid = state.parts[3];
    desktop_set_item_selection(selected_uid);
    desktop_set_item_view_loading();
    desktop_refresh({loading: "item"});
}

function desktop_render_feed(feed, selected_item_uid) {
    desktop_elements.feedTitle.textContent = feed.name;
    desktop_set_read_button(desktop_elements.feedReadButton, feed,
        {loading: desktop_set_item_list_loading, refresh: {loading: "items"}},
    );
    desktop_set_content(desktop_elements.itemList, gen_item_list(feed, {
        selected_uid: selected_item_uid,
        item_url: item => `/feeds/${feed.uid}/items/${item.uid}`,
        link_href: item => item.url,
        link_handler: desktop_select_item,
    }));
}

let desktop_feeds = [];

async function desktop_refresh(opts = {}) {
    if (opts.loading === "items") {
        desktop_set_item_list_loading();
    } else if (opts.loading === "item") {
        desktop_set_item_view_loading();
    } else if (opts.loading !== false) {
        desktop_set_loading();
    }

    let state = get_current_state();
    if (state.login) {
        try {
            let response = await login(state.token);
            if (response.status === 200) {
                history.replaceState(null, "", HOME_PATH);
                await desktop_refresh();
            } else {
                desktop_set_error("Please login.");
            }
        } catch (err) {
            desktop_set_error(`Unexpected error logging in: ${err}`);
        }
        return;
    }

    let selected_uid = state.parts[1] || "all";
    if (state.parts[0] !== "feeds") {
        desktop_set_error("Please login.");
        return;
    }

    let feeds;
    if (opts.feed_switch && desktop_feeds.length) {
        feeds = desktop_feeds;
    } else {
        let feeds_response;
        try {
            [feeds_response, feeds] = await fetch_feeds();
        } catch (err) {
            desktop_set_error(`Unexpected error fetching feed list: ${err}`);
            return;
        }
        if (feeds_response.status !== 200) {
            desktop_set_error(response_error(feeds_response, feeds, "Feed not found."));
            return;
        }
    }

    let status_page = selected_uid === "status";
    if (status_page) {
        desktop_render_feed_list(feeds, selected_uid);
        desktop_elements.feedTitle.textContent = "Status";
        desktop_set_read_button(desktop_elements.feedReadButton);
        desktop_set_content(desktop_elements.itemList, gen_status_content(feeds, {
            link_handler: desktop_refresh,
        }));
        desktop_set_item_content();
        return;
    }

    let feed_summary = feeds.find(feed => feed.uid === selected_uid);
    if (!feed_summary) {
        desktop_set_error("Feed not found.");
        return;
    }

    let feed_response, feed;
    try {
        [feed_response, feed] = await fetch_feed(selected_uid);
    } catch (err) {
        desktop_set_error(`Unexpected error fetching feed: ${err}`);
        return;
    }
    if (feed_response.status !== 200) {
        desktop_set_error(response_error(feed_response, feed, "Feed not found."));
        return;
    }

    let item_uid = state.parts[2] === "items" ? state.parts[3] : undefined;
    let item;
    if (item_uid) {
        let item_summary = feed.items?.find(item => item.uid === item_uid);
        let item_feed_uid = item_summary?.feed_uid || selected_uid;
        let item_response, item_data;
        try {
            [item_response, item_data] = await fetch_item(item_feed_uid, item_uid);
        } catch (err) {
            desktop_set_error(`Unexpected error fetching item: ${err}`);
            return;
        }
        if (item_response.status !== 200) {
            desktop_set_error(response_error(item_response, item_data, "Item not found."));
            return;
        }
        item = item_data;

        if (!item.read && await mark_item_as_read(item.feed_uid, item.uid)) {
            item.read = true;
            desktop_update_read_state(feeds, feed, item);
        }
    }

    if (!opts.feed_switch) {
        desktop_render_feed_list(feeds, selected_uid);
    } else {
        desktop_set_feed_selection(selected_uid);
    }
    desktop_render_feed(feed, item_uid);
    desktop_set_item_content(item);
}

async function desktop_start() {
    if (document.readyState === "loading") {
        window.addEventListener("load", () => desktop_start().catch(start_mobile_mode), {once: true});
        return;
    }
    if (!is_desktop_mode()) {
        unload_desktop_assets();
        start_mobile_mode();
        return;
    }
    desktop_init();
    start_mode_detection();
    history.scrollRestoration = "manual";

    window.onpopstate = () => desktop_refresh();

    let state = get_current_state();
    if (!state.login && state.parts[0] !== "feeds") {
        history.replaceState(null, "", HOME_PATH);
    } else if (!state.login && !state.parts[1]) {
        history.replaceState(null, "", HOME_PATH);
    }
    return desktop_refresh();
}
