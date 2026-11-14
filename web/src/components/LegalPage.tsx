import type { ReactNode } from "react";
import { site } from "@/lib/site";

/** Plain, readable layout for the privacy policy and terms. */
export function LegalPage({ title, children }: { title: string; children: ReactNode }) {
  return (
    <article className="max-w-3xl pb-16">
      <h1 className="border-b border-signal pb-4 text-4xl font-light text-signal sm:text-5xl">{title}</h1>
      <p className="mt-4 text-sm text-zinc-500">Last updated {site.legalUpdated}</p>
      <div className="mt-8 space-y-4 text-[16px] leading-7 text-zinc-700 [&_a]:text-signal [&_a]:underline [&_a]:underline-offset-4 [&_a:hover]:text-ink [&_h2]:mt-12 [&_h2]:text-2xl [&_h2]:font-light [&_h2]:text-signal [&_li]:pl-1 [&_strong]:font-semibold [&_strong]:text-ink [&_ul]:list-disc [&_ul]:space-y-2 [&_ul]:pl-6">
        {children}
      </div>
    </article>
  );
}

export function Contact() {
  return (
    <a href={`mailto:${site.email}`}>{site.email}</a>
  );
}
