import React from "react";

export const MailToDesktopButton = () => {
    const subject = encodeURIComponent("Check out Socle");
    const body = encodeURIComponent(
        "Check out Socle, a CLI tool for managing stacked Git branches.\n\nDownload using brew:\nbrew install benekuehn/tap/socle\n\nOr visit: https://github.com/benekuehn/socle",
    );
    const mailtoLink = `mailto:?subject=${subject}&body=${body}`;

    return (
        <a
            href={mailtoLink}
            className='md:hidden flex items-center justify-center rounded-lg border border-zinc-800 px-6 py-3 text-sm font-medium text-zinc-100 hover:bg-zinc-900 hover:border-zinc-900 transition-colors'
        >
            Mail to desktop
        </a>
    );
};
