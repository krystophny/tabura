const targetDefinitions = {
  slopshell_circle_dot: {
    category: 'circle',
    platforms: {
      web: '#slopshell-circle-dot',
      ios: 'slopshell_circle_dot',
      android: 'slopshell_circle_dot',
    },
  },
  slopshell_circle_segment_pointer: {
    category: 'circle',
    platforms: {
      web: '[data-segment="pointer"]',
      ios: 'slopshell_circle_pointer',
      android: 'slopshell_circle_pointer',
    },
  },
  slopshell_circle_segment_highlight: {
    category: 'circle',
    platforms: {
      web: '[data-segment="highlight"]',
      ios: 'slopshell_circle_highlight',
      android: 'slopshell_circle_highlight',
    },
  },
  slopshell_circle_segment_ink: {
    category: 'circle',
    platforms: {
      web: '[data-segment="ink"]',
      ios: 'slopshell_circle_ink',
      android: 'slopshell_circle_ink',
    },
  },
  slopshell_circle_segment_text_note: {
    category: 'circle',
    platforms: {
      web: '[data-segment="text_note"]',
      ios: 'slopshell_circle_text_note',
      android: 'slopshell_circle_text_note',
    },
  },
  slopshell_circle_segment_prompt: {
    category: 'circle',
    platforms: {
      web: '[data-segment="prompt"]',
      ios: 'slopshell_circle_prompt',
      android: 'slopshell_circle_prompt',
    },
  },
  slopshell_circle_segment_dialogue: {
    category: 'circle',
    platforms: {
      web: '[data-segment="dialogue"]',
      ios: 'slopshell_circle_dialogue',
      android: 'slopshell_circle_dialogue',
    },
  },
  slopshell_circle_segment_meeting: {
    category: 'circle',
    platforms: {
      web: '[data-segment="meeting"]',
      ios: 'slopshell_circle_meeting',
      android: 'slopshell_circle_meeting',
    },
  },
  slopshell_circle_segment_silent: {
    category: 'circle',
    platforms: {
      web: '[data-segment="silent"]',
      ios: 'slopshell_circle_silent',
      android: 'slopshell_circle_silent',
    },
  },
  canvas_viewport: {
    category: 'circle',
    platforms: {
      web: '#canvas-viewport',
      ios: 'canvas_viewport',
      android: 'canvas_viewport',
    },
  },
  indicator_border: {
    category: 'indicator',
    platforms: {
      web: '#indicator-border',
      ios: 'indicator_border',
      android: 'indicator_border',
    },
  },
  indicator_simulate_recording: {
    category: 'indicator',
    kind: 'test_hook',
    platforms: {
      web: '#indicator-simulate-recording',
    },
  },
  indicator_simulate_working: {
    category: 'indicator',
    kind: 'test_hook',
    platforms: {
      web: '#indicator-simulate-working',
    },
  },
  indicator_override_clear: {
    category: 'indicator',
    kind: 'test_hook',
    platforms: {
      web: '#indicator-override-clear',
    },
  },
};

const requiredCoverageTargets = [
  'slopshell_circle_dot',
  'slopshell_circle_segment_pointer',
  'slopshell_circle_segment_highlight',
  'slopshell_circle_segment_ink',
  'slopshell_circle_segment_text_note',
  'slopshell_circle_segment_prompt',
  'slopshell_circle_segment_dialogue',
  'slopshell_circle_segment_meeting',
  'slopshell_circle_segment_silent',
  'canvas_viewport',
  'indicator_border',
];

const requiredIndicatorStates = ['idle', 'listening', 'paused', 'recording', 'working'];

function getTargetDefinition(target) {
  return targetDefinitions[target] || null;
}

function getTargetPlatforms(target) {
  const definition = getTargetDefinition(target);
  return definition ? Object.keys(definition.platforms) : [];
}

function getWebSelector(target) {
  const definition = getTargetDefinition(target);
  return definition && definition.platforms.web ? definition.platforms.web : null;
}

module.exports = {
  getTargetDefinition,
  getTargetPlatforms,
  getWebSelector,
  requiredCoverageTargets,
  requiredIndicatorStates,
  targetDefinitions,
};
