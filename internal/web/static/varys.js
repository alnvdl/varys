const TOKEN_PREFIX = "token:";
const HOME_PATH = "/feeds/all";

function get_current_state() {
    let url = window.location.pathname;
    let hash = window.location.hash;

    if (hash) {
        hash = hash.substr(1);
        return {
            login: hash.startsWith(TOKEN_PREFIX),
            token: hash.substr(TOKEN_PREFIX.length),
            parts: hash.split("/").slice(1)
        }
    }

    return {
        login: false,
        token: "",
        parts: url.split("/").slice(1)
    }
}

async function fetch_json(url) {
    let rsp = await fetch(url);
    let data = await rsp.json();
    return [rsp, data];
}

async function fetch_feeds() {
    return fetch_json("/api/feeds");
}

async function fetch_feed(uid) {
    return fetch_json(`/api/feeds/${uid}`);
}

async function fetch_item(fuid, iuid) {
    return fetch_json(`/api/feeds/${fuid}/items/${iuid}`);
}

async function login(token) {
    return fetch("/login", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({token})
    });
}

async function mark_feed_as_read(fuid, before) {
    let rsp = await fetch(`/api/feeds/${fuid}/read`, {
        method: "POST",
        body: JSON.stringify({before}),
        headers: {"Content-Type": "application/json"},
    });
    return rsp.status === 200;
}

async function mark_item_as_read(fuid, iuid) {
    let rsp = await fetch(`/api/feeds/${fuid}/items/${iuid}/read`, {method: "POST"});
    return rsp.status === 200;
}

async function mark_feed_as_read_and_refresh(fuid, before) {
    set_loading();
    await mark_feed_as_read(fuid, before);
    refresh();
}

async function mark_all_feeds_as_read_and_refresh(feed) {
    set_loading();
    await mark_feed_as_read(feed.uid, feed.last_updated);
    refresh();
}

// Page state.
function error(err) {
    console.trace(err);
    let div = document.createElement("div");
    div.setAttribute("class", "error");
    div.appendChild(document.createTextNode(err));
    set_content(div);
    reset_controls({breadcrumbs: true});
}

async function show_feeds() {
    let rsp, data;
    try {
        reset_controls({breadcrumbs: true, read_button: true});
        set_loading();
        [rsp, data] = await fetch_feeds();
    } catch (err) {
        error(`Unexpected error fetching feed list: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            let all_feed = data.find(feed => feed.uid === "all");
            reset_controls({
                breadcrumbs: true,
                read_button: () => mark_all_feeds_as_read_and_refresh(all_feed),
            });
            let feed_list = document.createElement("ol");
            feed_list.setAttribute("class", "feed-list")
            data.forEach(feed => {
                let li = document.createElement("li");

                let a = document.createElement("a");
                a.appendChild(document.createTextNode(feed.name));
                link(a, `/feeds/${feed.uid}`);

                let unread = feed.item_count - feed.read_count;
                if (unread) {
                    let span = document.createElement("span");
                    span.setAttribute("class", "feed-unread-count")
                    span.appendChild(document.createTextNode(unread));
                    a.appendChild(span);
                } else {
                    li.setAttribute("class", "feed-read")
                }

                li.appendChild(a);
                feed_list.appendChild(li);
            });

            let li = document.createElement("li");
            let a = document.createElement("a");
            a.setAttribute("class", "feed-status")
            a.appendChild(document.createTextNode("Status"));
            link(a, "/feeds/status");
            li.appendChild(a);
            feed_list.appendChild(li);

            set_content(feed_list);
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

async function show_feed(uid) {
    let rsp, data;
    try {
        reset_controls({breadcrumbs: true, read_button: true});
        set_loading();
        [rsp, data] = await fetch_feed(uid);
    } catch (err) {
        error(`Unexpected error fetching feed: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 404:
            error("Feed not found.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            reset_controls({
                breadcrumbs: {uid: data.uid, name: data.name},
                read_button: () => mark_feed_as_read_and_refresh(data.uid, data.last_updated),
            });
            let item_list = document.createElement("div");
            item_list.classList = "item-list";
            data.items?.forEach(item => {
                item_list.appendChild(gen_item(item, {list_view: true}));
            });
            set_content(item_list);
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

async function show_item(fuid, iuid) {
    let rsp, data;

    try {
        set_loading();
        [rsp, data] = await fetch_item(fuid, iuid);
    } catch (err) {
        error(`Unexpected error fetching item: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 404:
            error("Item not found.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            reset_controls({
                breadcrumbs: {uid: data.feed, name: data.feed_name},
                open_button: data.url,
            });
            set_content(gen_item(data));
            window.scrollTo(0, 0);
            if (!data.read) {
                mark_item_as_read(data.feed_uid, data.uid);
            }
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

function gen_item(item, opts) {
    let list_view = (opts && opts.list_view) || false;

    let itemDiv = document.createElement("div");
    itemDiv.classList.add("item");

    let headerDiv = document.createElement("div");
    headerDiv.classList.add("item-header");
    itemDiv.appendChild(headerDiv);;

    let titleDiv = document.createElement("div");
    titleDiv.classList.add("item-title");
    headerDiv.appendChild(titleDiv);;

    let detailsDiv = document.createElement("div");
    detailsDiv.classList.add("item-details");
    headerDiv.appendChild(detailsDiv);;

    let contentDiv = document.createElement("div");
    contentDiv.classList.add("item-content");
    itemDiv.append(contentDiv);

    let title = document.createTextNode(item.title);
    titleDiv.appendChild(title);
    if (!list_view) {
        titleDiv.classList.add("item-title-bold");
    }

    let details = [];
    details.push(item.feed_name);

    if (item.authors) {
        if (item.authors.length > 32) {
            item.authors = item.authors.substr(0, 32) + "...";
        }
        details.push("by " + item.authors);
    }

    let when = relative_time_desc(item.timestamp);
    details.push(when);

    let detailsNode = document.createTextNode(details.join(" · "));
    detailsDiv.appendChild(detailsNode);

    if (!list_view) {
        contentDiv.innerHTML = item.content;
        itemDiv.setAttribute("class", "item-full");
    } else {
        itemDiv.setAttribute("class", "item-summary");
        contentDiv.parentNode.removeChild(contentDiv);
    }

    if (list_view) {
        let a = document.createElement("a");
        link(a, `/feeds/${item.feed_uid}/items/${item.uid}`);
        a.setAttribute("class", "item-link");
        a.appendChild(itemDiv);
        itemDiv = a;
        if (item.read) {
            a.setAttribute("class", a.getAttribute("class") + " item-link-read");
        }
    }

    return itemDiv;
}

async function refresh() {
    let state = get_current_state();
    if (state.login) {
        let rsp = await login(state.token);
        if (rsp.status === 200) {
            history.replaceState(null, "", HOME_PATH);
            refresh();
        } else {
            error("Please login.");
        }
        return;
    }

    if (state.parts[0] === "feeds" && state.parts[2] === "items") {
        await show_item(state.parts[1], state.parts[3]);
    } else if (state.parts[0] === "feeds" && state.parts[1] === "status") {
        await show_status_feeds();
    } else if (state.parts[0] === "feeds") {
        if (state.parts[1]) {
            await show_feed(state.parts[1]);
        } else {
            await show_feeds();
        }
    } else {
        error("Please login.");
    }
}

async function show_status_feeds() {
    let rsp, data;
    set_loading();
    try {
        [rsp, data] = await fetch_feeds();
    } catch (err) {
        error(`Unexpected error fetching feed list: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            reset_controls({breadcrumbs: true});
            let container = document.createElement("div");
            const THIRTY_DAYS = 30 * 24 * 60 * 60;
            data.forEach(feed => {
                if (feed.uid === "all") return; // Skip the 'all' feed
                let feedDiv = document.createElement("div");
                feedDiv.classList.add("feed-status-block");

                let status = "";
                if (feed.last_error && feed.last_error.trim() !== "") {
                    status = "🔴";
                } else if (feed.last_updated && (
                    (now() - feed.last_item) > THIRTY_DAYS ||
                    feed.item_count === 0) ||
                    feed.last_item === 0) {
                    status = "🟡";
                } else {
                    status = "🟢";
                }

                let name = document.createElement("div");
                name.classList.add("feed-status-name");
                let a = document.createElement("a");
                link(a, `/feeds/${feed.uid}`);
                a.textContent = status + " " + feed.name;
                name.appendChild(a);
                feedDiv.appendChild(name);

                let table = document.createElement("table");
                table.classList.add("feed-status-table");

                let urlRow = document.createElement("tr");
                let urlLabel = document.createElement("td");
                urlLabel.textContent = "URL";
                let urlValue = document.createElement("td");
                urlValue.innerHTML = `<code>${feed.url || ""}</code>`;
                urlRow.appendChild(urlLabel);
                urlRow.appendChild(urlValue);
                table.appendChild(urlRow);

                let itemsRow = document.createElement("tr");
                let itemsLabel = document.createElement("td");
                itemsLabel.textContent = "Items";
                let itemsValue = document.createElement("td");
                itemsValue.textContent = `${feed.item_count} total, ${feed.item_count - feed.read_count} unread`;
                itemsRow.appendChild(itemsLabel);
                itemsRow.appendChild(itemsValue);
                table.appendChild(itemsRow);

                let updatedRow = document.createElement("tr");
                let updatedLabel = document.createElement("td");
                updatedLabel.textContent = "Last update";
                let updatedValue = document.createElement("td");
                updatedValue.textContent = feed.last_updated ? relative_time_desc(feed.last_updated) : "";
                updatedRow.appendChild(updatedLabel);
                updatedRow.appendChild(updatedValue);
                table.appendChild(updatedRow);

                if (feed.last_item) {
                    let recentRow = document.createElement("tr");
                    let recentLabel = document.createElement("td");
                    recentLabel.textContent = "Last item";
                    let recentValue = document.createElement("td");
                    recentValue.textContent = relative_time_desc(feed.last_item);
                    recentRow.appendChild(recentLabel);
                    recentRow.appendChild(recentValue);
                    table.appendChild(recentRow);
                }

                if (feed.last_error) {
                    let errorRow = document.createElement("tr");
                    let errorLabel = document.createElement("td");
                    errorLabel.textContent = "Error";
                    let errorValue = document.createElement("td");
                    errorValue.textContent = feed.last_error ? feed.last_error : "none";
                    errorRow.appendChild(errorLabel);
                    errorRow.appendChild(errorValue);
                    table.appendChild(errorRow);
                }

                feedDiv.appendChild(table);
                container.appendChild(feedDiv);
            });

            set_content(container);
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

function set_content(...content) {
    let elem = document.querySelector("#content");
    while (elem.firstChild) {
        elem.removeChild(elem.lastChild);
    }
    content.forEach(child => elem.appendChild(child));
}

function set_loading() {
    let div = document.createElement("div");
    div.classList.add("loading");
    let spinner = document.createElement("div");
    spinner.classList.add("spinner");
    div.appendChild(spinner);
    // iOS flashes an enlarged scrollbar in the loading screen when the
    // scrollbar is still being displayed because of a recent scroll event. The
    // following code makes us overlay the loading screen on iOS to prevent
    // that from happening.
    let isIOS = /iPad|iPhone|iPod/.test(navigator.platform);
    if (isIOS) {
        div.classList.add("loading-ios");
        let elem = document.querySelector("#content");
        window.scrollTo(0, 0);
        elem.childNodes.forEach(node => {
            node.classList.add("invisible");
        })
        elem.appendChild(div);
    } else {
        set_content(div);
    }
}

function link(a, url) {
    a.setAttribute("href", url);
    if (a.onclick) return;
    a.onclick = e => {
        history.pushState(null, "", a.href);
        refresh();
        e.preventDefault();
    };
}

function reset_controls(config) {
    if (!config) {
        document.querySelector("#controls").classList.add("hidden");
        return;
    }

    document.querySelector("#controls").classList.remove("hidden");

    let breadcrumbs = document.querySelector(`#breadcrumbs`);
    breadcrumbs.classList.add("hidden");
    if (config.breadcrumbs) {
        breadcrumbs.classList.remove("hidden");

        let items = document.querySelector("#breadcrumb-items");
        items.textContent = "";

        let bitem = document.createElement("div");
        bitem.setAttribute("class", "breadcrumb-item");
        a = document.createElement("a");
        link(a, "/feeds");
        a.appendChild(document.createTextNode("Feeds"));
        bitem.appendChild(a);
        items.appendChild(bitem);

        if (typeof config.breadcrumbs === 'object' && config.breadcrumbs !== null) {
            bitem = document.createElement("li");
            bitem.setAttribute("class", "breadcrumb-item");
            a = document.createElement("a");
            link(a, `/feeds/${config.breadcrumbs.uid}`);
            let name = config.breadcrumbs.name;
            a.appendChild(document.createTextNode(`${name}`));
            bitem.appendChild(a);
            items.appendChild(bitem);
        }
    }

    let read_button = document.querySelector(`#read-button`);
    read_button.classList.add("hidden");
    read_button.onclick = null;
    if (config.read_button) {
        read_button.classList.remove("hidden");
        if (typeof config.read_button === "function") {
            read_button.onclick = config.read_button;
        }
    }

    let open_button = document.querySelector(`#open-button`);
    open_button.classList.add("hidden");
    if (config.open_button) {
        link(open_button, config.open_button);
        open_button.setAttribute("target", "_blank");
        open_button.setAttribute("rel", "noopener noreferrer");
        open_button.classList.remove("hidden");
    }
}

function start() {
    history.scrollRestoration = "auto";
    reset_controls();
    refresh();
};

window.onpopstate = refresh;

window.onload = () => {
    let read_button = document.querySelector("#read-button");
    read_button.addEventListener("touchstart", () => { });

    let open_button = document.querySelector("#open-button");
    open_button.addEventListener("touchstart", () => { });

    start();
};

// now returns the current time in seconds since the Unix epoch.
function now() {
    return Math.floor(new Date() / 1000);
}

// seconds_ago receives a timestamp and returns the number of seconds that have
// passed since the given timestamp. It returns a negative number if the given
// timestamp is in the future.
function seconds_ago(timestamp) {
    return now() - timestamp;
}

const seconds_in_a_minute = 60;
const seconds_in_an_hour = 60 * seconds_in_a_minute;
const seconds_in_a_day = 24 * seconds_in_an_hour;
const seconds_in_a_week = 7 * seconds_in_a_day;
const seconds_in_a_month = 30 * seconds_in_a_day;
const seconds_in_a_year = 365 * seconds_in_a_day;

const units = [
    [seconds_in_a_minute, "minute"],
    [seconds_in_an_hour, "hour"],
    [seconds_in_a_day, "day"],
    [seconds_in_a_week, "week"],
    [seconds_in_a_month, "month"],
    [seconds_in_a_year, "year"],
]

const rtf = new Intl.RelativeTimeFormat("en", {
    localeMatcher: "best fit",
    numeric: "auto",
    style: "long"
});

// relative_time_desc receives a timestamp and returns a human-readable string
// in English representing the time difference between the given timestamp and
// the current time (e.g., 3 hours ago).
function relative_time_desc(timestamp) {
    let diff = -seconds_ago(timestamp);

    let chosen_quantity = 1;
    let chosen_unit = "second";
    for (let i = 0; i < units.length; i++) {
        let [quantity, unit] = units[i];
        if (Math.abs(diff) < quantity) {
            break
        }
        chosen_quantity = quantity;
        chosen_unit = unit;
    }

    let adapted_diff = Math.round(diff / chosen_quantity);
    return rtf.format(adapted_diff, chosen_unit);
}
