import Image from "next/image";
import Link from "next/link";
import { site } from "@/lib/site";

const link = "border-b border-signal pb-px text-white hover:text-signal";
const heading = "text-[17px] font-semibold text-signal";

/** Laid out like the footer on the school website: crest, then four columns with blue headings. */
export function Footer() {
  return (
    <footer className="mt-10 bg-gradient-to-r from-[#14203c] to-[#00225a] text-[16px] leading-7 text-white print:hidden">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-14 sm:px-6 md:grid-cols-[90px_repeat(4,1fr)] md:gap-8">
        <div>
          <Image src="/logo.png" alt="King's High School crest" width={474} height={471} className="h-[72px] w-auto [filter:drop-shadow(0_0_1px_#fff)_drop-shadow(0_0_1px_#fff)]" />
        </div>

        <section>
          <h2 className={heading}>Athletics Day</h2>
          <ul className="mt-5 space-y-1">
            <li>
              <Link href="/schedule" className={link}>
                Schedule
              </Link>
            </li>
            <li>
              <Link href="/progress" className={link}>
                Day progress
              </Link>
            </li>
            <li>
              <Link href="/login" className={link}>
                Staff sign in
              </Link>
            </li>
          </ul>
        </section>

        <section>
          <h2 className={heading}>School</h2>
          <address className="mt-5 not-italic">
            {site.school}
            <br />
            270 Bay View Road
            <br />
            Dunedin 9012
          </address>
          <p className="mt-4">
            <a href="https://www.kingshigh.school.nz" className={link}>
              School website
            </a>
          </p>
        </section>

        <section>
          <h2 className={heading}>Legal</h2>
          <ul className="mt-5 space-y-1">
            <li>
              <Link href="/privacy" className={link}>
                Privacy policy
              </Link>
            </li>
            <li>
              <Link href="/terms" className={link}>
                Terms of service
              </Link>
            </li>
          </ul>
        </section>

        <section>
          <h2 className={heading}>Website help</h2>
          <p className="mt-5">
            Problems with this website, or questions about your information:
          </p>
          <p className="mt-2">
            {site.operator}
            <br />
            <a href={`mailto:${site.email}`} className={link}>
              {site.email}
            </a>
          </p>
        </section>
      </div>
      <p className="pb-8 text-center text-xs font-semibold">Copyright &copy; {new Date().getFullYear()} {site.school}</p>
    </footer>
  );
}
