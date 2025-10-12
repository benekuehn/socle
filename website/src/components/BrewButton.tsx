import { CopyButton } from "./CopyButton";

export const BrewButton = ({ className }: { className?: string }) => {
    return (
        <div
            className={`flex items-center text-sm text-zinc-100 bg-zinc-900 rounded-md px-3 py-2 gap-3 ${className}`}
        >
            <code>brew install benekuehn/tap/socle</code>
            <CopyButton
                text='brew install benekuehn/tap/socle'
                ariaLabel='Copy brew install command'
            />
        </div>
    );
};
