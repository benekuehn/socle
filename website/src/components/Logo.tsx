interface LogoProps {
  width?: number;
  height?: number;
  className?: string;
}

export function Logo({ width = 32, height = 32, className = '' }: LogoProps) {
  return (
    <svg 
      width={width} 
      height={height} 
      viewBox="0 0 392 256" 
      fill="none" 
      xmlns="http://www.w3.org/2000/svg" 
      className={className}
    >
      <path d="M390.998 111.714L196 225.167L1.00098 111.714L196 0.575195L390.998 111.714Z" fill="url(#paint0_linear_1_122)" stroke="url(#paint1_linear_1_122)"/>
      <path d="M0 111.709L196 225.745V256L0 141.963V111.709Z" fill="url(#paint2_linear_1_122)"/>
      <path d="M196 256L392 141.964V111.709L196 225.745V256Z" fill="url(#paint3_linear_1_122)"/>
      <path d="M344.331 85.1152L196 171.418L47.6677 85.1162L196 0.575195L344.331 85.1152Z" fill="url(#paint4_linear_1_122)" stroke="url(#paint5_linear_1_122)"/>
      <path d="M46.6666 85.1113L196 171.996V195.047L46.6666 108.162V85.1113Z" fill="url(#paint6_linear_1_122)"/>
      <path d="M196 195.047L345.333 108.163V85.1115L196 171.996V195.047Z" fill="url(#paint7_linear_1_122)"/>
      <path d="M196 0L303.333 61.174L196 123.622L88.6667 61.174L196 0Z" fill="url(#paint8_linear_1_122)"/>
      <path d="M88.6667 61.1738L196 123.622V140.19L88.6667 77.7417V61.1738Z" fill="url(#paint9_linear_1_122)"/>
      <path d="M196 140.191L303.333 77.742V61.1741L196 123.623V140.191Z" fill="url(#paint10_linear_1_122)"/>
      <defs>
        <linearGradient id="paint0_linear_1_122" x1="196" y1="0" x2="196" y2="257" gradientUnits="userSpaceOnUse">
          <stop stopColor="#8C8C8C"/>
          <stop offset="1" stopColor="#2A2A2A"/>
        </linearGradient>
        <linearGradient id="paint1_linear_1_122" x1="196" y1="0" x2="196" y2="225.745" gradientUnits="userSpaceOnUse">
          <stop stopColor="#444444"/>
          <stop offset="0.153846" stopOpacity="0"/>
        </linearGradient>
        <linearGradient id="paint2_linear_1_122" x1="98" y1="111.709" x2="98" y2="256" gradientUnits="userSpaceOnUse">
          <stop stopColor="#252525"/>
          <stop offset="1" stopColor="#ACACAC"/>
        </linearGradient>
        <linearGradient id="paint3_linear_1_122" x1="196" y1="-3.51021e-05" x2="304.5" y2="190.5" gradientUnits="userSpaceOnUse">
          <stop stopColor="#8C8C8C"/>
          <stop offset="1" stopColor="#2A2A2A"/>
        </linearGradient>
        <linearGradient id="paint4_linear_1_122" x1="196" y1="0" x2="196" y2="257.5" gradientUnits="userSpaceOnUse">
          <stop stopColor="#9C9C9C"/>
          <stop offset="1" stopColor="#2A2A2A"/>
        </linearGradient>
        <linearGradient id="paint5_linear_1_122" x1="196" y1="0" x2="196" y2="171.997" gradientUnits="userSpaceOnUse">
          <stop stopColor="#444444"/>
          <stop offset="0.153846" stopOpacity="0"/>
        </linearGradient>
        <linearGradient id="paint6_linear_1_122" x1="121.333" y1="85.1113" x2="121.333" y2="195.047" gradientUnits="userSpaceOnUse">
          <stop stopColor="#252525"/>
          <stop offset="1" stopColor="#ACACAC"/>
        </linearGradient>
        <linearGradient id="paint7_linear_1_122" x1="314" y1="175.5" x2="196" y2="1.00007" gradientUnits="userSpaceOnUse">
          <stop stopColor="#2A2A2A"/>
          <stop offset="1" stopColor="#9C9C9C"/>
        </linearGradient>
        <linearGradient id="paint8_linear_1_122" x1="196" y1="0" x2="196" y2="257" gradientUnits="userSpaceOnUse">
          <stop stopColor="#ACACAC"/>
          <stop offset="1" stopColor="#2A2A2A"/>
        </linearGradient>
        <linearGradient id="paint9_linear_1_122" x1="142.333" y1="61.1738" x2="142.333" y2="140.19" gradientUnits="userSpaceOnUse">
          <stop stopColor="#252525"/>
          <stop offset="1" stopColor="#ACACAC"/>
        </linearGradient>
        <linearGradient id="paint10_linear_1_122" x1="284.5" y1="181" x2="196.37" y2="0.709889" gradientUnits="userSpaceOnUse">
          <stop stopColor="#2A2A2A"/>
          <stop offset="1" stopColor="#ACACAC"/>
        </linearGradient>
      </defs>
    </svg>
  );
} 